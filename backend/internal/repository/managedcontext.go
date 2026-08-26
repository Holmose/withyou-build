package repository

import (
	"context"
	"time"

	domain "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/managedcontext"
)

// ManagedContextRepository defines managed context persistence capability.
type ManagedContextRepository interface {
	UpsertContextState(ctx context.Context, item *domain.ChatUpstreamContextState) error
	GetContextState(ctx context.Context, conversationID, upstreamID, userID uint) (*domain.ChatUpstreamContextState, error)
	GetContextStateByConversationID(ctx context.Context, conversationID uint) (*domain.ChatUpstreamContextState, error)
	ListContextStates(ctx context.Context, offset, limit int, upstreamID *uint, userID *uint, state *string) ([]domain.ChatUpstreamContextState, int64, error)
	CountSessionsByState(ctx context.Context, upstreamID uint) (map[string]int64, error)
	CreateMetricEvent(ctx context.Context, item *domain.ChatContextMetricEvent) error
	MetricEventExists(ctx context.Context, upstreamID uint, requestID string) (bool, error)
	MetricEventExistsByQueuedAt(ctx context.Context, upstreamID uint, queuedAt int64) (bool, error)
	ListMetricEvents(ctx context.Context, offset, limit int, upstreamID *uint, conversationID *uint, createdFrom, createdTo *time.Time) ([]domain.ChatContextMetricEvent, int64, error)
	GetLatestMetricEventByRequestID(ctx context.Context, upstreamID uint, requestID string) (*domain.ChatContextMetricEvent, error)
	MetricsSummary(ctx context.Context, upstreamID *uint, from, to *time.Time) (*domain.ContextMetricsSummary, error)
	ListUpstreamsWithContextInfo(ctx context.Context, offset, limit int, stateFilter string) ([]ManagedContextUpstreamRow, int64, error)
}

// ContextBuild holds upstream runtime build identity (wire DTO for admin API).
type ContextBuild struct {
	Commit  string `json:"commit"`
	Time    string `json:"time"`
	Version string `json:"version"`
}

// ManagedContextUpstreamRow is the joined read model for the upstream list
// display; lives here (not in domain) because it carries JSON wire contracts.
type ManagedContextUpstreamRow struct {
	ID                           uint           `json:"id"`
	Name                         string         `json:"name"`
	BaseURL                      string         `json:"baseURL"`
	ContextStrategy              domain.ContextStrategy `json:"contextStrategy"`
	ContextProtocolVersion       string         `json:"contextProtocolVersion"`
	ContextBuild                 *ContextBuild  `json:"contextBuild,omitempty"`
	ManagedContextRolloutState   domain.RolloutState    `json:"managedContextRolloutState"`
	ContextCapabilitiesJSON      string         `json:"contextCapabilitiesJSON"`
	ContextCapabilitiesCheckedAt *time.Time     `json:"contextCapabilitiesCheckedAt,omitempty"`
	ContextCapabilitiesError     string         `json:"contextCapabilitiesError"`
	ActiveSessions               int64          `json:"activeSessions"`
	PendingMigration             int64          `json:"pendingMigration"`
}
