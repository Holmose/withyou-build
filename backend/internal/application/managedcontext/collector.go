package managedcontext

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	domain "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/managedcontext"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"go.uber.org/zap"
)

const (
	defaultCollectInterval = 60 * time.Second
	llmRequestsEndpoint    = "/admin/llm-requests"
)

// withyouLLMRequestsResponse mirrors WITHYOU /admin/llm-requests payload.
type withyouLLMRequestsResponse struct {
	Pool struct {
		MaxConcurrent int `json:"maxConcurrent"`
	} `json:"pool"`
	Queued         []withyouTraceEntry          `json:"queued"`
	Inflight       []withyouTraceEntry          `json:"inflight"`
	ActiveRequests []withyouTraceEntry          `json:"active_requests"`
	Recent         []withyouTraceEntry          `json:"recent"`
	ContextMetrics []withyouContextMetricEvent `json:"context_metrics"`
}

// withyouTraceEntry mirrors WITHYOU request-tracer recent entries.
type withyouTraceEntry struct {
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	ConversationID string `json:"conversationId"`
	SessionID      string `json:"sessionId"`
	Provider       string `json:"provider"`
	Model          string `json:"model"`
	MessagesCount  int    `json:"messagesCount"`
	EstTokens      int64  `json:"estTokens"`
	Status         string `json:"status"`
	DurationMs     int64  `json:"durationMs"`
	Error          string `json:"error"`
	QueuedAt       int64  `json:"queuedAt"`
	StartedAt      int64  `json:"startedAt"`
}

// withyouContextMetricEvent mirrors WITHYOU context-request-metric final fields.
type withyouContextMetricEvent struct {
	RequestID       string  `json:"requestId"`
	ContextMode     string  `json:"contextMode"`
	EpochHash       string  `json:"epochHash"`
	Status          string  `json:"status"`
	ErrorCode       string  `json:"errorCode"`
	TotalMs         int64   `json:"totalMs"`
	ProtocolVersion float64 `json:"protocolVersion"`
	CreatedAt       int64   `json:"createdAt"`
	Stored          *struct {
		MessageCount    int   `json:"messageCount"`
		EstimatedTokens int64 `json:"estimatedTokens"`
	} `json:"stored"`
	Merged *struct {
		MessageCount    int   `json:"messageCount"`
		EstimatedTokens int64 `json:"estimatedTokens"`
	} `json:"merged"`
	Final *struct {
		MessageCount    int   `json:"messageCount"`
		EstimatedTokens int64 `json:"estimatedTokens"`
		BudgetTokens    int64 `json:"budgetTokens"`
	} `json:"final"`
	Provider *struct {
		Provider             string `json:"provider"`
		Model                string `json:"model"`
		Status               string `json:"status"`
		ErrorCode            string `json:"errorCode"`
		EstimatedInputTokens int64  `json:"estimatedInputTokens"`
		ActualInputTokens    int64  `json:"actualInputTokens"`
		ActualOutputTokens   int64  `json:"actualOutputTokens"`
		DurationMs           int64  `json:"durationMs"`
	} `json:"provider"`
	Recall *struct {
		Executed bool            `json:"executed"`
		HitCount int             `json:"hitCount"`
		Degraded bool            `json:"degraded"`
		Hits     json.RawMessage `json:"hits"`
	} `json:"recall"`
	Compaction *struct {
		Count int `json:"count"`
	} `json:"compaction"`
	Trim *struct {
		TrimmedMessages int `json:"trimmedMessages"`
		TrimmedTokens   int `json:"trimmedTokens"`
	} `json:"trim"`
	Epoch *struct {
		Current int    `json:"current"`
		Hash    string `json:"hash"`
	} `json:"epoch"`
}

// RunMetricsCollector starts a background loop that pulls per-upstream request
// metrics from WITHYOU /admin/llm-requests into chat_context_metric_events.
func (s *Service) RunMetricsCollector(ctx context.Context) {
	s.log.Info("metrics_collector_started")
	interval := defaultCollectInterval
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.collectOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.collectOnce(ctx)
		}
	}
}

func (s *Service) collectOnce(ctx context.Context) {
	upstreams, _, err := s.channelRepo.ListUpstreams(ctx, repository.ListChannelUpstreamsInput{Limit: 1000})
	if err != nil {
		s.log.Warn("metrics_collector list_upstreams_failed", zap.Error(err))
		return
	}
	s.log.Info("metrics_collector_tick", zap.Int("upstreams", len(upstreams)))

	for _, u := range upstreams {
		if u.BaseURL == "" {
			continue
		}
		s.collectUpstream(ctx, u.ID, u.BaseURL, u.HeadersJSON, u.APIKeysEnc)
	}
}

func (s *Service) collectUpstream(ctx context.Context, upstreamID uint, baseURL, headersJSON, encryptedKeys string) {
	headers := parseCustomHeaders(headersJSON)
	if headers.Get("x-admin-token") == "" {
		s.log.Warn("metrics_collector no_admin_token", zap.Uint("upstream_id", upstreamID))
		return
	}

	if apiKey := s.selectAPIKeyOrEmpty(encryptedKeys); apiKey != "" {
		headers.Set("Authorization", "Bearer "+apiKey)
	}

	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	base = strings.TrimSuffix(base, "/v1")
	url := base + llmRequestsEndpoint
	s.log.Info("metrics_collector_fetch", zap.Uint("upstream_id", upstreamID), zap.String("url", url))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		s.log.Warn("metrics_collector request_error", zap.Uint("upstream_id", upstreamID), zap.Error(err))
		return
	}
	for k := range headers {
		req.Header.Set(k, headers.Get(k))
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.log.Warn("metrics_collector fetch_failed", zap.Uint("upstream_id", upstreamID), zap.String("url", url), zap.Error(err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		s.log.Warn("metrics_collector non_200", zap.Uint("upstream_id", upstreamID), zap.Int("status", resp.StatusCode), zap.String("body", string(body)))
		return
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		s.log.Warn("metrics_collector read_failed", zap.Uint("upstream_id", upstreamID), zap.Error(err))
		return
	}

	var payload withyouLLMRequestsResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		s.log.Warn("metrics_collector parse_failed", zap.Uint("upstream_id", upstreamID), zap.Error(err))
		return
	}

	cmByReq := make(map[string]withyouContextMetricEvent, len(payload.ContextMetrics))
	for _, cm := range payload.ContextMetrics {
		cmByReq[cm.RequestID] = cm
	}

	// context_metrics entries carry protocol/token/mode detail but key on the
	// DEEIX request id, not the in-app tracer id. recent entries carry the
	// conversation binding. Pair them by closeness of timestamps so each recent
	// entry can inherit real token counts, protocol version and context mode.
	var cmByCreated []withyouContextMetricEvent
	for _, cm := range payload.ContextMetrics {
		if cm.CreatedAt > 0 {
			cmByCreated = append(cmByCreated, cm)
		}
	}
	sort.Slice(cmByCreated, func(i, j int) bool { return cmByCreated[i].CreatedAt < cmByCreated[j].CreatedAt })

	var written int
	for _, entry := range payload.Recent {
		// kind 空 = 旧版上游（不过滤保持兼容）；chat = 对话主请求；
		// compact/compact_bg/rollup/memory 等为上游内部调用，不是用户请求。
		if entry.Kind != "" && entry.Kind != "chat" {
			continue
		}
		if entry.ID == "" {
			continue
		}
		exists, err := s.mcRepo.MetricEventExistsByQueuedAt(ctx, upstreamID, entry.QueuedAt)
		if err != nil || exists {
			continue
		}

		conversationID := parseConversationID(entry.ConversationID, entry.SessionID)
		userID := parseUserID(entry.SessionID)
		status := entry.Status
		errorCode := entry.Error
		latencyMS := entry.DurationMs
		msgCount := entry.MessagesCount
		estTokens := entry.EstTokens
		contextMode := "legacy"
		protocolVersion := ""
		sentTokens := int64(0)
		budgetTokens := int64(0)
		storedCount := 0
		mergedCount := 0
		recallCount := 0
		recallDegraded := false
		var recallHits *withyouRecallHits
		compactionCount := 0
		trimMessages := 0
		trimTokens := 0
		epochHash := ""

		if cm, ok := cmByReq[entry.ID]; ok {
			inheritContextMetric(&cm, &status, &errorCode, &latencyMS, &msgCount, &estTokens, &contextMode, &protocolVersion, &sentTokens)
			inheritContextDetail(&cm, &budgetTokens, &storedCount, &mergedCount, &recallCount, &recallDegraded, &recallHits, &trimMessages, &trimTokens, &epochHash)
			if cm.Compaction != nil && cm.Compaction.Count > 0 {
				compactionCount = cm.Compaction.Count
			}
		} else if match := matchContextMetric(cmByCreated, entry); match != nil {
			inheritContextMetric(match, &status, &errorCode, &latencyMS, &msgCount, &estTokens, &contextMode, &protocolVersion, &sentTokens)
			inheritContextDetail(match, &budgetTokens, &storedCount, &mergedCount, &recallCount, &recallDegraded, &recallHits, &trimMessages, &trimTokens, &epochHash)
			if match.Compaction != nil && match.Compaction.Count > 0 {
				compactionCount = match.Compaction.Count
			}
		}

		if errorCode == "success" {
			errorCode = ""
		}

		// WITHYOU's context_metrics always report contextMode=legacy (hardcoded
		// in its metric builder). The real mode DEEIX sent lives in the ack it
		// persisted to chat_upstream_context_states — use that as the truth.
		if contextMode == "" || contextMode == "legacy" {
			if state, err := s.mcRepo.GetContextStateByConversationID(ctx, uint(conversationID)); err == nil && state != nil && strings.TrimSpace(state.Mode) != "" {
				contextMode = state.Mode
			}
		}

		ev := &domain.ChatContextMetricEvent{
			RequestID:       entry.ID,
			ConversationID:  uint(conversationID),
			UpstreamID:      upstreamID,
			UserID:          userID,
			ContextMode:     contextMode,
			ProtocolVersion: protocolVersion,
			EpochHash:       epochHash,
			MessageCount:    msgCount,
			EstimatedTokens: estTokens,
			SentTokens:      sentTokens,
			BudgetTokens:    budgetTokens,
			StoredCount:     storedCount,
			MergedCount:     mergedCount,
			RecallCount:     recallCount,
			RecallDegraded:  recallDegraded,
			RecallHitsJSON:  recallHitsJSON(recallHits),
			CompactionCount: compactionCount,
			TrimMessages:    trimMessages,
			TrimTokens:      trimTokens,
			QueuedAt:        entry.QueuedAt,
			Status:          status,
			ErrorCode:       strings.TrimSpace(errorCode),
			LatencyMS:       latencyMS,
		}
		if err := s.mcRepo.CreateMetricEvent(ctx, ev); err != nil {
			s.log.Warn("metrics_collector write_failed", zap.Uint("upstream_id", upstreamID), zap.String("request_id", entry.ID), zap.Error(err))
			continue
		}
		written++
	}

	if written > 0 {
		s.log.Info("metrics_collector_written",
			zap.Uint("upstream_id", upstreamID),
			zap.Int("written", written),
			zap.Int("recent", len(payload.Recent)),
		)
	} else {
		s.log.Info("metrics_collector_noop",
			zap.Uint("upstream_id", upstreamID),
			zap.Int("recent", len(payload.Recent)),
			zap.Int("context_metrics", len(payload.ContextMetrics)),
		)
	}
}

// inheritContextMetric copies whichever context_metric detail is present onto
// the event being written. recent is always the primary source (it carries the
// conversation binding); context_metric only fills fields recent lacks.
func inheritContextMetric(cm *withyouContextMetricEvent, status, errorCode *string, latencyMS *int64, msgCount *int, estTokens *int64, contextMode, protocolVersion *string, sentTokens *int64) {
	if cm.Status != "" {
		*status = cm.Status
	}
	if cm.ErrorCode != "" {
		*errorCode = cm.ErrorCode
	}
	if cm.TotalMs > 0 {
		*latencyMS = cm.TotalMs
	}
	if cm.ContextMode != "" {
		*contextMode = cm.ContextMode
	}
	if cm.ProtocolVersion > 0 {
		*protocolVersion = strconv.Itoa(int(cm.ProtocolVersion))
	}
	if cm.Merged != nil && cm.Merged.MessageCount > 0 {
		*msgCount = cm.Merged.MessageCount
	}
	if cm.Final != nil && cm.Final.MessageCount > 0 {
		*msgCount = cm.Final.MessageCount
		// 裁剪后的最终上下文，而非合并前的大值
		if cm.Final.EstimatedTokens > 0 {
			*estTokens = cm.Final.EstimatedTokens
		}
	}
	if cm.Provider != nil && cm.Provider.ActualInputTokens > 0 {
		*sentTokens = cm.Provider.ActualInputTokens
		if *status == "" {
			*status = cm.Provider.Status
		}
	}
}

// inheritContextDetail copies the observability detail fields (context budget,
// stored/merged message counts, memory recall) onto the event being written.
// These come only from context_metrics; recent entries never carry them.
func inheritContextDetail(cm *withyouContextMetricEvent, budgetTokens *int64, storedCount, mergedCount, recallCount *int, recallDegraded *bool, recallHits **withyouRecallHits, trimMessages, trimTokens *int, epochHash *string) {
	if cm.Final != nil && cm.Final.BudgetTokens > 0 {
		*budgetTokens = cm.Final.BudgetTokens
	}
	if cm.Stored != nil {
		*storedCount = cm.Stored.MessageCount
	}
	if cm.Merged != nil {
		*mergedCount = cm.Merged.MessageCount
	}
	if cm.Recall != nil {
		*recallCount = cm.Recall.HitCount
		*recallDegraded = cm.Recall.Degraded
		if hits := parseRecallHits(cm.Recall.Hits); hits != nil {
			*recallHits = hits
		}
	}
	if cm.Trim != nil {
		*trimMessages = cm.Trim.TrimmedMessages
		*trimTokens = cm.Trim.TrimmedTokens
	}
	if cm.Epoch != nil && strings.TrimSpace(cm.Epoch.Hash) != "" {
		*epochHash = strings.TrimSpace(cm.Epoch.Hash)
	}
}

// withyouRecallHitDetail is the v2 recall hit payload: per-category evidence
// entry count and estimated injected material tokens. Legacy payloads stored
// plain booleans per category; parseRecallHits normalizes both.
type withyouRecallHitDetail struct {
	Count  int     `json:"count"`
	Tokens float64 `json:"tokens"`
}

type withyouRecallHits map[string]withyouRecallHitDetail

func parseRecallHits(raw json.RawMessage) *withyouRecallHits {
	if len(raw) == 0 || string(raw) == "null" || string(raw) == "{}" {
		return nil
	}
	var detail withyouRecallHits
	if err := json.Unmarshal(raw, &detail); err == nil && len(detail) > 0 {
		return &detail
	}
	var legacy map[string]bool
	if err := json.Unmarshal(raw, &legacy); err == nil {
		out := withyouRecallHits{}
		for k, v := range legacy {
			if v {
				out[k] = withyouRecallHitDetail{Count: 1}
			}
		}
		if len(out) > 0 {
			return &out
		}
	}
	return nil
}

// recallHitsJSON serializes which memory categories a request actually recalled,
// or an empty object when the upstream reported no hits detail.
func recallHitsJSON(hits *withyouRecallHits) string {
	if hits == nil {
		return "{}"
	}
	b, err := json.Marshal(hits)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// matchContextMetric pairs a tracer entry to the context_metric that started
// nearest to it (same request appears in both arrays, keyed differently).
func matchContextMetric(cmByCreated []withyouContextMetricEvent, entry withyouTraceEntry) *withyouContextMetricEvent {
	if entry.QueuedAt <= 0 || len(cmByCreated) == 0 {
		return nil
	}
	best := sort.Search(len(cmByCreated), func(i int) bool { return cmByCreated[i].CreatedAt >= entry.QueuedAt })
	candidates := []int{best}
	if best > 0 {
		candidates = append(candidates, best-1)
	}
	if best < len(cmByCreated) {
		candidates = append(candidates, best)
	}
	if best > 0 && best < len(cmByCreated) {
		candidates = append(candidates, best+1)
	}

	var closest *withyouContextMetricEvent
	minGap := int64(1 << 62)
	for _, idx := range candidates {
		if idx < 0 || idx >= len(cmByCreated) {
			continue
		}
		gap := cmByCreated[idx].CreatedAt - entry.QueuedAt
		if gap < 0 {
			gap = -gap
		}
		if gap < minGap {
			minGap = gap
			c := cmByCreated[idx]
			closest = &c
		}
	}
	if minGap <= 10*time.Second.Milliseconds() {
		return closest
	}
	return nil
}

// parseConversationID resolves a conversation id from the trace-entry
// conversationId field, falling back to WITHYOU's legacy session array
// ["withyou-session-v1", tenantId, userId, conversationId].
func parseConversationID(conversationID, sessionID string) uint {
	if id, err := strconv.ParseUint(strings.TrimSpace(conversationID), 10, 32); err == nil {
		return uint(id)
	}
	if len(sessionID) >= 2 && sessionID[0] == '[' {
		var parts []string
		if err := json.Unmarshal([]byte(sessionID), &parts); err == nil && len(parts) >= 4 {
			if id, err := strconv.ParseUint(strings.TrimSpace(parts[3]), 10, 32); err == nil {
				return uint(id)
			}
		}
	}
	if strings.Contains(sessionID, ",") {
		seg := strings.Split(sessionID, ",")
		if len(seg) >= 4 {
			if id, err := strconv.ParseUint(unquoteValue(seg[3]), 10, 32); err == nil {
				return uint(id)
			}
		}
	}
	return 0
}

// parseUserID resolves the user id from WITHYOU's session array
// ["withyou-session-v1", tenantId, userId, conversationId].
func parseUserID(sessionID string) string {
	if len(sessionID) >= 2 && sessionID[0] == '[' {
		var parts []string
		if err := json.Unmarshal([]byte(sessionID), &parts); err == nil && len(parts) >= 3 {
			return unquoteValue(parts[2])
		}
	}
	if strings.Contains(sessionID, ",") {
		seg := strings.Split(sessionID, ",")
		if len(seg) >= 3 {
			return unquoteValue(seg[2])
		}
	}
	return ""
}

// unquoteValue strips JSON quotes a value may carry when it reached us through
// the comma-split fallback (e.g. '"101"' from '["withyou-session-v1",...,"101",...]').
func unquoteValue(value string) string {
	return strings.Trim(value, `"`+" ")
}

// selectAPIKeyOrEmpty decrypts and returns first active key, or empty string on any error.
func (s *Service) selectAPIKeyOrEmpty(encrypted string) string {
	key, err := s.selectAPIKey(context.Background(), encrypted)
	if err != nil {
		return ""
	}
	return key
}

// parseCustomHeaders parses an upstream HeadersJSON map into an http.Header.
func parseCustomHeaders(raw string) http.Header {
	headers := http.Header{}
	if strings.TrimSpace(raw) == "" {
		return headers
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return headers
	}
	for k, v := range m {
		headers.Set(k, v)
	}
	return headers
}
