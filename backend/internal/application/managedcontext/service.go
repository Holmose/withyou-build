package managedcontext

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	domain "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/managedcontext"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/llm"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/pkg/secretbox"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"go.uber.org/zap"
)

type Service struct {
	mcRepo        repository.ManagedContextRepository
	channelRepo   repository.ChannelRepository
	auditService  AuditWriter
	encryptionKey string
	log           *zap.Logger
}

type AuditWriter interface {
	Write(ctx context.Context, requestID string, actorUserID uint, action, resource, resourceID, ip, userAgent string, detail interface{})
}

func NewService(mcRepo repository.ManagedContextRepository, channelRepo repository.ChannelRepository, auditService AuditWriter, encryptionKey string, log *zap.Logger) *Service {
	return &Service{
		mcRepo:        mcRepo,
		channelRepo:   channelRepo,
		auditService:  auditService,
		encryptionKey: encryptionKey,
		log:           log,
	}
}

// CapabilityCheckResult is the result of a capability check.
type CapabilityCheckResult struct {
	Success         bool
	ProtocolVersion string
	Build           WithYouBuild
	Features        WithYouFeatures
	Limits          WithYouLimits
	ErrorCode       string
	ErrorMessage    string
}

// WithYouFeatures from /v1/withyou/capabilities.
type WithYouFeatures struct {
	Bootstrap     bool `json:"bootstrap"`
	Delta         bool `json:"delta"`
	Epoch         bool `json:"epoch"`
	Idempotency   bool `json:"idempotency"`
	Metrics       bool `json:"metrics"`
	HistoryRecall bool `json:"history_recall"`
}

// WithYouLimits from /v1/withyou/capabilities.
type WithYouLimits struct {
	RequestBodyBytes        int `json:"request_body_bytes"`
	BootstrapTokens         int `json:"bootstrap_tokens"`
	ProviderInputTokens     int `json:"provider_input_tokens"`
	SingleMessageTokens     int `json:"single_message_tokens"`
	CompactionTriggerTokens int `json:"compaction_trigger_tokens"`
	CompactionTargetTokens  int `json:"compaction_target_tokens"`
	TailTurns               int `json:"tail_turns"`
	SummaryTokens           int `json:"summary_tokens"`
}

// WithYouBuild from /v1/withyou/capabilities.
type WithYouBuild struct {
	Commit  string `json:"commit"`
	Time    string `json:"time"`
	Version string `json:"version"`
}

// withYouCapabilitiesResponse is the raw response.
type withYouCapabilitiesResponse struct {
	ProtocolVersion string          `json:"protocol_version"`
	Build           WithYouBuild    `json:"build"`
	TokenEstimator  string          `json:"token_estimator"`
	Features        WithYouFeatures `json:"features"`
	Limits          WithYouLimits   `json:"limits"`
}

func (s *Service) CheckCapabilities(ctx context.Context, upstreamID uint, actorUserID uint, requestID, ip, userAgent string) (*CapabilityCheckResult, error) {
	upstream, err := s.channelRepo.GetUpstreamByID(ctx, upstreamID)
	if err != nil {
		return nil, fmt.Errorf("get upstream: %w", err)
	}

	apiKey, err := s.selectAPIKey(ctx, upstream.APIKeysEnc)
	if err != nil {
		return &CapabilityCheckResult{Success: false, ErrorCode: "NO_API_KEY", ErrorMessage: "no valid api key found"}, nil
	}

	url := llm.BuildVersionedEndpointURL(upstream.BaseURL, "v1", "/withyou/capabilities")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.updateUpstreamCapabilityError(ctx, upstreamID, "REQUEST_FAILED", err.Error())
		currentState := upstream.ManagedContextRolloutState
		if currentState == "" {
			currentState = string(domain.RolloutDisabled)
		}
		if currentState != string(domain.RolloutEnabled) && currentState != string(domain.RolloutMigrationPaused) && currentState != string(domain.RolloutDegraded) {
			s.channelRepo.UpdateManagedContextRolloutState(ctx, upstreamID, string(domain.RolloutCapabilityFailed))
		}
		return &CapabilityCheckResult{Success: false, ErrorCode: "REQUEST_FAILED", ErrorMessage: err.Error()}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	statusCode := resp.StatusCode

	s.log.Info("check_capabilities_response",
		zap.Uint("upstream_id", upstreamID),
		zap.String("upstream_name", upstream.Name),
		zap.String("base_url", upstream.BaseURL),
		zap.Int("status", statusCode),
		zap.String("body", string(body)),
		zap.String("url", url),
	)

	currentState := upstream.ManagedContextRolloutState
	if currentState == "" {
		currentState = string(domain.RolloutDisabled)
	}

	if statusCode == 401 {
		s.updateUpstreamCapabilityError(ctx, upstreamID, "AUTH_401", string(body))
		if currentState != string(domain.RolloutEnabled) && currentState != string(domain.RolloutMigrationPaused) && currentState != string(domain.RolloutDegraded) {
			s.channelRepo.UpdateManagedContextRolloutState(ctx, upstreamID, string(domain.RolloutCapabilityFailed))
		}
		return &CapabilityCheckResult{Success: false, ErrorCode: "AUTH_401", ErrorMessage: "unauthorized"}, nil
	}
	if statusCode == 403 {
		s.updateUpstreamCapabilityError(ctx, upstreamID, "AUTH_403", string(body))
		if currentState != string(domain.RolloutEnabled) && currentState != string(domain.RolloutMigrationPaused) && currentState != string(domain.RolloutDegraded) {
			s.channelRepo.UpdateManagedContextRolloutState(ctx, upstreamID, string(domain.RolloutCapabilityFailed))
		}
		return &CapabilityCheckResult{Success: false, ErrorCode: "AUTH_403", ErrorMessage: "forbidden"}, nil
	}
	if statusCode == 404 {
		s.updateUpstreamCapabilityError(ctx, upstreamID, "NOT_FOUND_404", string(body))
		if currentState != string(domain.RolloutEnabled) && currentState != string(domain.RolloutMigrationPaused) && currentState != string(domain.RolloutDegraded) {
			s.channelRepo.UpdateManagedContextRolloutState(ctx, upstreamID, string(domain.RolloutCapabilityFailed))
		}
		return &CapabilityCheckResult{Success: false, ErrorCode: "NOT_FOUND_404", ErrorMessage: "endpoint not found"}, nil
	}
	if statusCode >= 500 {
		s.updateUpstreamCapabilityError(ctx, upstreamID, fmt.Sprintf("SERVER_%d", statusCode), string(body))
		if currentState != string(domain.RolloutEnabled) && currentState != string(domain.RolloutMigrationPaused) && currentState != string(domain.RolloutDegraded) {
			s.channelRepo.UpdateManagedContextRolloutState(ctx, upstreamID, string(domain.RolloutCapabilityFailed))
		}
		return &CapabilityCheckResult{Success: false, ErrorCode: fmt.Sprintf("SERVER_%d", statusCode), ErrorMessage: "upstream server error"}, nil
	}
	if statusCode != 200 {
		s.updateUpstreamCapabilityError(ctx, upstreamID, fmt.Sprintf("HTTP_%d", statusCode), string(body))
		if currentState != string(domain.RolloutEnabled) && currentState != string(domain.RolloutMigrationPaused) && currentState != string(domain.RolloutDegraded) {
			s.channelRepo.UpdateManagedContextRolloutState(ctx, upstreamID, string(domain.RolloutCapabilityFailed))
		}
		return &CapabilityCheckResult{Success: false, ErrorCode: fmt.Sprintf("HTTP_%d", statusCode), ErrorMessage: fmt.Sprintf("unexpected status %d", statusCode)}, nil
	}

	var caps withYouCapabilitiesResponse
	if err := json.Unmarshal(body, &caps); err != nil {
		s.updateUpstreamCapabilityError(ctx, upstreamID, "INVALID_JSON", err.Error())
		if currentState != string(domain.RolloutEnabled) && currentState != string(domain.RolloutMigrationPaused) && currentState != string(domain.RolloutDegraded) {
			s.channelRepo.UpdateManagedContextRolloutState(ctx, upstreamID, string(domain.RolloutCapabilityFailed))
		}
		return &CapabilityCheckResult{Success: false, ErrorCode: "INVALID_JSON", ErrorMessage: "response is not valid JSON"}, nil
	}

	// Sanitize: only store what we need
	capsJSON, _ := json.Marshal(caps)
	checkedAt := time.Now()

	err = s.channelRepo.UpdateManagedContextFields(ctx, upstreamID, string(capsJSON), checkedAt, "")
	if err != nil {
		s.log.Warn("failed to update upstream capability fields", zap.Error(err))
	}

	// P0-4: legacy servers respond with no protocol_version — treat as capability check failure
	if caps.ProtocolVersion == "" {
		capsDump, _ := json.Marshal(caps)
		s.log.Warn("check_capabilities_legacy",
			zap.Uint("upstream_id", upstreamID),
			zap.String("upstream_name", upstream.Name),
			zap.String("base_url", upstream.BaseURL),
			zap.String("url", url),
			zap.String("caps", string(capsDump)),
		)
		s.updateUpstreamCapabilityError(ctx, upstreamID, "LEGACY_CAPABILITY", "server returned legacy capability format (no protocol_version)")
		if currentState != string(domain.RolloutEnabled) && currentState != string(domain.RolloutMigrationPaused) && currentState != string(domain.RolloutDegraded) {
			s.channelRepo.UpdateManagedContextRolloutState(ctx, upstreamID, string(domain.RolloutCapabilityFailed))
		}
		return &CapabilityCheckResult{Success: false, ErrorCode: "LEGACY_CAPABILITY", ErrorMessage: "upstream returned legacy capability format (no protocol_version)"}, nil
	}

	_ = s.channelRepo.UpdateManagedContextProtocolVersion(ctx, upstreamID, caps.ProtocolVersion)

	_ = s.channelRepo.UpdateManagedContextBuild(ctx, upstreamID, caps.Build.Commit, caps.Build.Time, caps.Build.Version)

	// ponytail: transition rollout state after successful capability check
	if currentState == string(domain.RolloutDisabled) || currentState == string(domain.RolloutChecking) || currentState == string(domain.RolloutCapabilityFailed) {
		s.channelRepo.UpdateManagedContextRolloutState(ctx, upstreamID, string(domain.RolloutCapabilityReady))
	}

	return &CapabilityCheckResult{
		Success:         true,
		ProtocolVersion: caps.ProtocolVersion,
		Build:           caps.Build,
		Features:        caps.Features,
		Limits:          caps.Limits,
	}, nil
}

func (s *Service) updateUpstreamCapabilityError(ctx context.Context, upstreamID uint, errorCode, errorMsg string) {
	checkedAt := time.Now()
	s.channelRepo.UpdateManagedContextFields(ctx, upstreamID, "{}", checkedAt, errorCode)
	// ponytail: do not override enabled/paused state on error
}

// selectAPIKey decrypts and selects first active key.
func (s *Service) selectAPIKey(ctx context.Context, encrypted string) (string, error) {
	raw, err := s.decryptAPIKeys(encrypted)
	if err != nil {
		return "", err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty key config")
	}

	var parsed struct {
		Strategy string `json:"strategy"`
		Keys     []struct {
			Key    string `json:"key"`
			Status string `json:"status"`
		} `json:"keys"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return "", err
	}
	for _, k := range parsed.Keys {
		if k.Status == "active" || k.Status == "" {
			return k.Key, nil
		}
	}
	if len(parsed.Keys) > 0 {
		return parsed.Keys[0].Key, nil
	}
	return "", fmt.Errorf("no active key")
}

func (s *Service) decryptAPIKeys(encrypted string) (string, error) {
	return secretbox.DecryptString(s.encryptionKey, encrypted)
}

func (s *Service) TestProtocol(ctx context.Context, upstreamID uint, actorUserID uint, requestID, ip, userAgent string) (*TestProtocolResult, error) {
	caps, err := s.CheckCapabilities(ctx, upstreamID, actorUserID, requestID, ip, userAgent)
	if err != nil {
		return nil, err
	}
	result := &TestProtocolResult{
		CapabilitiesResult: caps,
		ProbeResult:        &ProbeResult{},
	}

	upstream, _ := s.channelRepo.GetUpstreamByID(ctx, upstreamID)
	if upstream == nil {
		result.ProbeResult.ErrorCode = "UPSTREAM_NOT_FOUND"
		return result, nil
	}

	apiKey, err := s.selectAPIKey(ctx, upstream.APIKeysEnc)
	if err != nil {
		result.ProbeResult.ErrorCode = "NO_API_KEY"
		result.ProbeResult.ErrorMessage = err.Error()
		return result, nil
	}

	probeURL := strings.TrimSuffix(upstream.BaseURL, "/") + "/v1/chat/completions"
	probeBody := `{"model":"test","messages":[{"role":"user","content":"ping"}],"max_tokens":5}`
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, probeURL, bytes.NewBufferString(probeBody))
	if err != nil {
		result.ProbeResult.ErrorCode = "REQUEST_ERROR"
		result.ProbeResult.ErrorMessage = err.Error()
		return result, nil
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	latencyMS := time.Since(start).Milliseconds()

	result.ProbeResult.LatencyMS = latencyMS
	if err != nil {
		result.ProbeResult.ErrorCode = "REQUEST_FAILED"
		result.ProbeResult.ErrorMessage = err.Error()
		return result, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	result.ProbeResult.StatusCode = resp.StatusCode

	if resp.StatusCode >= 500 {
		result.ProbeResult.ErrorCode = fmt.Sprintf("SERVER_%d", resp.StatusCode)
		result.ProbeResult.ErrorMessage = string(body)
	} else if resp.StatusCode == 200 && !caps.Success {
		result.ProbeResult.ErrorCode = "SWALLOWED_ERROR"
		result.ProbeResult.ErrorMessage = "request returned 200 but capability check failed"
	} else if resp.StatusCode != 200 && caps.Success {
		result.ProbeResult.ErrorCode = fmt.Sprintf("HTTP_%d", resp.StatusCode)
		result.ProbeResult.ErrorMessage = string(body)
	}

	s.writeAudit(ctx, actorUserID, requestID, "managed_context_test", "upstream", strconv.FormatUint(uint64(upstreamID), 10), ip, userAgent, map[string]interface{}{
		"upstream_id":          upstreamID,
		"status_code":          resp.StatusCode,
		"latency_ms":           latencyMS,
		"capabilities_success": caps.Success,
	})

	return result, nil
}

type TestProtocolResult struct {
	CapabilitiesResult *CapabilityCheckResult `json:"capabilitiesResult"`
	ProbeResult        *ProbeResult           `json:"probeResult"`
}

type ProbeResult struct {
	StatusCode   int    `json:"statusCode"`
	LatencyMS    int64  `json:"latencyMS"`
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

func (s *Service) EnableUpstream(ctx context.Context, upstreamID uint, actorUserID uint, requestID, ip, userAgent string) error {
	upstream, err := s.channelRepo.GetUpstreamByID(ctx, upstreamID)
	if err != nil {
		return fmt.Errorf("get upstream: %w", err)
	}

	currentState := upstream.ManagedContextRolloutState
	if currentState == "" {
		currentState = string(domain.RolloutDisabled)
	}

	var newState string
	switch currentState {
	case string(domain.RolloutCapabilityReady):
		newState = string(domain.RolloutEnabled)
	case string(domain.RolloutDegraded):
		newState = string(domain.RolloutEnabled)
	case string(domain.RolloutMigrationPaused):
		newState = string(domain.RolloutEnabled)
	default:
		return fmt.Errorf("cannot enable from state %s", currentState)
	}

	if err := s.channelRepo.UpdateManagedContextRolloutState(ctx, upstreamID, newState); err != nil {
		return fmt.Errorf("update rollout state: %w", err)
	}

	s.writeAudit(ctx, actorUserID, requestID, "managed_context_enable", "upstream", strconv.FormatUint(uint64(upstreamID), 10), ip, userAgent, map[string]interface{}{
		"upstream_id": upstreamID,
		"old_state":   currentState,
		"new_state":   newState,
	})

	s.recordMetricEvent(ctx, upstreamID, "managed_context", "managed-context-v1", "", 0, 0, 0, 0, 0, 0, 0, "success", "")

	return nil
}

func (s *Service) PauseUpstream(ctx context.Context, upstreamID uint, actorUserID uint, requestID, ip, userAgent string) error {
	upstream, err := s.channelRepo.GetUpstreamByID(ctx, upstreamID)
	if err != nil {
		return fmt.Errorf("get upstream: %w", err)
	}

	currentState := upstream.ManagedContextRolloutState
	if currentState == "" {
		currentState = string(domain.RolloutDisabled)
	}

	newState := string(domain.RolloutMigrationPaused)

	if err := s.channelRepo.UpdateManagedContextRolloutState(ctx, upstreamID, newState); err != nil {
		return fmt.Errorf("update rollout state: %w", err)
	}

	s.writeAudit(ctx, actorUserID, requestID, "managed_context_pause", "upstream", strconv.FormatUint(uint64(upstreamID), 10), ip, userAgent, map[string]interface{}{
		"upstream_id": upstreamID,
		"old_state":   currentState,
		"new_state":   newState,
	})

	return nil
}

func (s *Service) DisableUpstream(ctx context.Context, upstreamID uint, actorUserID uint, requestID, ip, userAgent string) error {
	upstream, err := s.channelRepo.GetUpstreamByID(ctx, upstreamID)
	if err != nil {
		return fmt.Errorf("get upstream: %w", err)
	}

	currentState := upstream.ManagedContextRolloutState
	if currentState == "" {
		currentState = string(domain.RolloutDisabled)
	}

	newState := string(domain.RolloutDisabled)

	if err := s.channelRepo.UpdateManagedContextRolloutState(ctx, upstreamID, newState); err != nil {
		return fmt.Errorf("update rollout state: %w", err)
	}

	s.writeAudit(ctx, actorUserID, requestID, "managed_context_disable", "upstream", strconv.FormatUint(uint64(upstreamID), 10), ip, userAgent, map[string]interface{}{
		"upstream_id": upstreamID,
		"old_state":   currentState,
		"new_state":   newState,
	})

	return nil
}

func (s *Service) GetOverview(ctx context.Context, upstreamID *uint) (*OverviewData, error) {
	summary, err := s.mcRepo.MetricsSummary(ctx, upstreamID, nil, nil)
	if err != nil {
		return nil, err
	}

	rows, total, err := s.mcRepo.ListUpstreamsWithContextInfo(ctx, 0, 100, "")
	if err != nil {
		return nil, err
	}

	var activeUpstreams int64
	var enabledUpstreams int64
	for _, r := range rows {
		activeUpstreams++
		if r.ManagedContextRolloutState == domain.RolloutEnabled {
			enabledUpstreams++
		}
	}

	return &OverviewData{
		EnabledUpstreams: enabledUpstreams,
		ActiveSessions:   summary.ActiveSessions,
		PendingMigration: summary.PendingMigration,
		OverflowCount:    summary.ContextOverflowCount,
		P95LatencyMS:     summary.P95LatencyMS,
		Upstreams:        rows,
		TotalUpstreams:   total,
	}, nil
}

type OverviewData struct {
	EnabledUpstreams int64                              `json:"enabledUpstreams"`
	ActiveSessions   int64                              `json:"activeSessions"`
	PendingMigration int64                              `json:"pendingMigration"`
	OverflowCount    int64                              `json:"overflowCount"`
	P95LatencyMS     int64                              `json:"p95LatencyMS"`
	Upstreams        []repository.ManagedContextUpstreamRow `json:"upstreams"`
	TotalUpstreams   int64                              `json:"totalUpstreams"`
}

func (s *Service) GetMetrics(ctx context.Context, upstreamID *uint, conversationID *uint, from, to *time.Time, page, pageSize int) ([]MetricEventRow, int64, error) {
	offset := (page - 1) * pageSize
	events, total, err := s.mcRepo.ListMetricEvents(ctx, offset, pageSize, upstreamID, conversationID, from, to)
	if err != nil {
		return nil, 0, err
	}
	rows := make([]MetricEventRow, 0, len(events))
	for _, e := range events {
		rows = append(rows, MetricEventRow{
			ID:              e.ID,
			RequestID:       e.RequestID,
			ConversationID:  e.ConversationID,
			UpstreamID:      e.UpstreamID,
			UserID:          e.UserID,
			ContextMode:     e.ContextMode,
			ProtocolVersion: e.ProtocolVersion,
			EpochHash:       e.EpochHash,
			MessageCount:    e.MessageCount,
			EstimatedTokens: e.EstimatedTokens,
			SentTokens:      e.SentTokens,
			BudgetTokens:    e.BudgetTokens,
			StoredCount:     e.StoredCount,
			MergedCount:     e.MergedCount,
			BootstrapCount:  e.BootstrapCount,
			DeltaCount:      e.DeltaCount,
			CompactionCount: e.CompactionCount,
			TrimMessages:    e.TrimMessages,
			TrimTokens:      e.TrimTokens,
			RecallCount:     e.RecallCount,
			RecallDegraded:  e.RecallDegraded,
			RecallHitsJSON:  e.RecallHitsJSON,
			Status:          e.Status,
			ErrorCode:       e.ErrorCode,
			LatencyMS:       e.LatencyMS,
			CreatedAt:       e.CreatedAt.Format(time.RFC3339),
		})
	}
	return rows, total, nil
}

type MetricEventRow struct {
	ID              uint   `json:"id"`
	RequestID       string `json:"requestID"`
	ConversationID  uint   `json:"conversationID"`
	UpstreamID      uint   `json:"upstreamID"`
	UserID          string `json:"userID"`
	ContextMode     string `json:"contextMode"`
	ProtocolVersion string `json:"protocolVersion"`
	EpochHash       string `json:"epochHash"`
	MessageCount    int    `json:"messageCount"`
	EstimatedTokens int64  `json:"estimatedTokens"`
	SentTokens      int64  `json:"sentTokens"`
	BudgetTokens    int64  `json:"budgetTokens"`
	StoredCount     int    `json:"storedCount"`
	MergedCount     int    `json:"mergedCount"`
	BootstrapCount  int    `json:"bootstrapCount"`
	DeltaCount      int    `json:"deltaCount"`
	CompactionCount int    `json:"compactionCount"`
	TrimMessages    int    `json:"trimMessages"`
	TrimTokens      int    `json:"trimTokens"`
	RecallCount     int    `json:"recallCount"`
	RecallDegraded  bool   `json:"recallDegraded"`
	RecallHitsJSON  string `json:"recallHitsJSON"`
	Status          string `json:"status"`
	ErrorCode       string `json:"errorCode"`
	LatencyMS       int64  `json:"latencyMS"`
	CreatedAt       string `json:"createdAt"`
}

func (s *Service) ListSessions(ctx context.Context, upstreamID *uint, state *string, page, pageSize int) ([]SessionRow, int64, error) {
	offset := (page - 1) * pageSize
	states, total, err := s.mcRepo.ListContextStates(ctx, offset, pageSize, upstreamID, nil, state)
	if err != nil {
		return nil, 0, err
	}
	rows := make([]SessionRow, 0, len(states))
	for _, st := range states {
		var lastBootstrapAt, lastDeltaAt, updatedAt string
		if st.LastBootstrapAt != nil {
			lastBootstrapAt = st.LastBootstrapAt.Format(time.RFC3339)
		}
		if st.LastDeltaAt != nil {
			lastDeltaAt = st.LastDeltaAt.Format(time.RFC3339)
		}
		if !st.UpdatedAt.IsZero() {
			updatedAt = st.UpdatedAt.Format(time.RFC3339)
		}
		rows = append(rows, SessionRow{
			ID:                      st.ID,
			ConversationID:          st.ConversationID,
			UpstreamID:              st.UpstreamID,
			UserID:                  st.UserID,
			SessionID:               st.SessionID,
			State:                   string(st.State),
			Mode:                    st.Mode,
			Epoch:                   st.Epoch,
			EpochHash:               st.EpochHash,
			BranchLeafMessageID:     st.BranchLeafMessageID,
			LastRequestID:           st.LastRequestID,
			LastBootstrapAt:         lastBootstrapAt,
			LastDeltaAt:             lastDeltaAt,
			LastErrorCode:           st.LastErrorCode,
			UpdatedAt:               updatedAt,
			UpstreamName:            st.UpstreamName,
			UpstreamProtocolVersion: st.ContextProtocolVersion,
			UpstreamBuild: func() *WithYouBuild {
				if st.ContextBuildCommit == "" && st.ContextBuildTime == "" && st.ContextBuildVersion == "" {
					return nil
				}
				return &WithYouBuild{Commit: st.ContextBuildCommit, Time: st.ContextBuildTime, Version: st.ContextBuildVersion}
			}(),
		})
	}
	return rows, total, nil
}

type SessionRow struct {
	ID                      uint          `json:"id"`
	ConversationID          uint          `json:"conversationID"`
	UpstreamID              uint          `json:"upstreamID"`
	UserID                  uint          `json:"userID"`
	SessionID               string        `json:"sessionID"`
	State                   string        `json:"state"`
	Mode                    string        `json:"mode"`
	Epoch                   int           `json:"epoch"`
	EpochHash               string        `json:"epochHash"`
	BranchLeafMessageID     uint          `json:"branchLeafMessageID"`
	LastRequestID           string        `json:"lastRequestID"`
	LastBootstrapAt         string        `json:"lastBootstrapAt"`
	LastDeltaAt             string        `json:"lastDeltaAt"`
	LastErrorCode           string        `json:"lastErrorCode"`
	UpdatedAt               string        `json:"updatedAt"`
	UpstreamName            string        `json:"upstreamName"`
	UpstreamProtocolVersion string        `json:"upstreamProtocolVersion"`
	UpstreamBuild           *WithYouBuild `json:"upstreamBuild"`
}

func (s *Service) ResyncConversation(ctx context.Context, conversationID, actorUserID uint, requestID, ip, userAgent string) error {
	state, err := s.mcRepo.GetContextStateByConversationID(ctx, conversationID)
	if err != nil && err != repository.ErrNotFound {
		return fmt.Errorf("get context state: %w", err)
	}

	epochHash := fmt.Sprintf("resync-%d-%d", conversationID, time.Now().UnixNano())
	now := time.Now()
	ns := &domain.ChatUpstreamContextState{
		ConversationID: conversationID,
		UpstreamID:     0,
		UserID:         actorUserID,
		State:          domain.StateResyncPending,
		EpochHash:      epochHash,
		LastErrorCode:  "",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if state != nil {
		ns.ID = state.ID
		ns.UpstreamID = state.UpstreamID
		ns.UserID = state.UserID
		ns.BranchLeafMessageID = state.BranchLeafMessageID
		ns.LastRequestID = state.LastRequestID
	}
	if ns.UpstreamID == 0 && state != nil {
		ns.UpstreamID = state.UpstreamID
	}

	err = s.mcRepo.UpsertContextState(ctx, ns)
	if err != nil {
		return fmt.Errorf("upsert context state: %w", err)
	}

	s.writeAudit(ctx, actorUserID, requestID, "managed_context_resync", "conversation", strconv.FormatUint(uint64(conversationID), 10), ip, userAgent, map[string]interface{}{
		"conversation_id": conversationID,
		"new_state":       domain.StateResyncPending,
		"epoch_hash":      epochHash,
	})

	return nil
}

func (s *Service) writeAudit(ctx context.Context, actorUserID uint, requestID, action, resource, resourceID, ip, userAgent string, detail interface{}) {
	if s.auditService != nil {
		s.auditService.Write(ctx, requestID, actorUserID, action, resource, resourceID, ip, userAgent, detail)
	}
}

func (s *Service) recordMetricEvent(ctx context.Context, upstreamID uint, mode, protocol, epochHash string, msgCount int, estTokens, sentTokens int64, bootstrap, delta, compaction, recall int, status, errorCode string) {
	ev := &domain.ChatContextMetricEvent{
		RequestID:       "",
		ConversationID:  0,
		UpstreamID:      upstreamID,
		ContextMode:     mode,
		ProtocolVersion: protocol,
		EpochHash:       epochHash,
		MessageCount:    msgCount,
		EstimatedTokens: estTokens,
		SentTokens:      sentTokens,
		BootstrapCount:  bootstrap,
		DeltaCount:      delta,
		CompactionCount: compaction,
		RecallCount:     recall,
		Status:          status,
		ErrorCode:       errorCode,
		LatencyMS:       0,
	}
	s.mcRepo.CreateMetricEvent(ctx, ev)
}
