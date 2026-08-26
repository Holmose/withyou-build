package managedcontext

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	domain "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/managedcontext"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
)

// RealtimeEntry is one live WithYou pipeline request (queued or inflight).
type RealtimeEntry struct {
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	ConversationID string `json:"conversationId"`
	Status         string `json:"status"`
	QueuedAt       int64  `json:"queuedAt"`
	StartedAt      int64  `json:"startedAt"` // W2 预留：WithYou 填充后前端按处理起点计时
	DurationMs     int64  `json:"durationMs"`
}

// RealtimeStatus is the live snapshot shown by the admin realtime panel.
type RealtimeStatus struct {
	// ActiveRequests 是 WithYou 处理管线中的 chat 请求（LLM 调用前的盲区段），
	// 完成/失败即消失；与网关 inflight（llm-N 调用中）并存。
	ActiveRequests []RealtimeEntry `json:"activeRequests"`
	Inflight       []RealtimeEntry `json:"inflight"`
	Queued         []RealtimeEntry `json:"queued"`
	Recent         []RealtimeEntry `json:"recent"`
	InflightByKind map[string]int  `json:"inflightByKind"`
	MaxConcurrent  int             `json:"maxConcurrent"`
	RecentStats    struct {
		Total      int64 `json:"total"`
		Success    int64 `json:"success"`
		SampleSize int   `json:"sampleSize"`
	} `json:"recentStats"`
	FetchedAt int64 `json:"fetchedAt"`
}

func newMetricEventRow(e domain.ChatContextMetricEvent) MetricEventRow {
	return MetricEventRow{
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
	}
}

// GetMetricEventByRequest resolves one tracer request id to its newest stored
// metric event row, for the realtime panel's click-through detail dialog.
func (s *Service) GetMetricEventByRequest(ctx context.Context, upstreamID uint, requestID string) (*MetricEventRow, error) {
	event, err := s.mcRepo.GetLatestMetricEventByRequestID(ctx, upstreamID, requestID)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, nil
	}
	row := newMetricEventRow(*event)
	return &row, nil
}

// GetRealtimeStatus pulls the live request pipeline from one WithYou upstream
// and merges it with rolling success stats computed from local metric events.
func (s *Service) GetRealtimeStatus(ctx context.Context, upstreamID uint) (*RealtimeStatus, error) {
	out := &RealtimeStatus{ActiveRequests: []RealtimeEntry{}, Inflight: []RealtimeEntry{}, Queued: []RealtimeEntry{}, Recent: []RealtimeEntry{}, InflightByKind: map[string]int{}}

	upstreams, _, err := s.channelRepo.ListUpstreams(ctx, repository.ListChannelUpstreamsInput{Limit: 1000})
	if err != nil {
		return nil, err
	}
	var target *repository.ChannelUpstreamListRow
	for i := range upstreams {
		if upstreams[i].ID == upstreamID {
			target = &upstreams[i]
			break
		}
	}
	if target == nil || strings.TrimSpace(target.BaseURL) == "" {
		return out, nil
	}

	headers := parseCustomHeaders(target.HeadersJSON)
	if headers.Get("x-admin-token") == "" {
		return out, nil
	}
	base := strings.TrimRight(strings.TrimSpace(target.BaseURL), "/")
	base = strings.TrimSuffix(base, "/v1")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+llmRequestsEndpoint, nil)
	if err != nil {
		return out, nil
	}
	for k := range headers {
		req.Header.Set(k, headers.Get(k))
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return out, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return out, nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return out, nil
	}

	var payload withyouLLMRequestsResponse
	if json.Unmarshal(body, &payload) != nil {
		return out, nil
	}

	toEntry := func(e withyouTraceEntry) RealtimeEntry {
		return RealtimeEntry{
			ID:             e.ID,
			Kind:           e.Kind,
			ConversationID: e.ConversationID,
			Status:         e.Status,
			QueuedAt:       e.QueuedAt,
			StartedAt:      e.StartedAt,
			DurationMs:     e.DurationMs,
		}
	}
	for _, e := range payload.ActiveRequests {
		out.ActiveRequests = append(out.ActiveRequests, toEntry(e))
	}
	for _, e := range payload.Inflight {
		entry := toEntry(e)
		kind := entry.Kind
		if kind == "" {
			kind = "chat"
		}
		out.InflightByKind[kind]++
		out.Inflight = append(out.Inflight, entry)
	}
	for _, e := range payload.Queued {
		out.Queued = append(out.Queued, toEntry(e))
	}
	// 最近请求：WithYou 环形缓冲（内存态，重启清零），最多 20 条供面板展示。
	for i, e := range payload.Recent {
		if i >= 20 {
			break
		}
		out.Recent = append(out.Recent, toEntry(e))
	}
	out.MaxConcurrent = payload.Pool.MaxConcurrent

	events, _, err := s.mcRepo.ListMetricEvents(ctx, 0, 50, &upstreamID, nil, nil, nil)
	if err == nil {
		out.RecentStats.SampleSize = len(events)
		for _, e := range events {
			if e.Status == "success" {
				out.RecentStats.Success++
			}
			out.RecentStats.Total++
		}
	}
	out.FetchedAt = time.Now().UnixMilli()
	return out, nil
}
