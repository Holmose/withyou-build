package managedcontext

import "time"

type ContextState string

const (
	StateLegacy           ContextState = "legacy"
	StateBootstrapPending ContextState = "bootstrap_pending"
	StateBootstrapping    ContextState = "bootstrapping"
	StateDeltaActive      ContextState = "delta_active"
	StateDeltaFailed      ContextState = "delta_failed"
	StateResyncPending    ContextState = "resync_pending"
	StateBootstrapFailed  ContextState = "bootstrap_failed"
)

type RolloutState string

const (
	RolloutDisabled          RolloutState = "disabled"
	RolloutChecking          RolloutState = "checking"
	RolloutCapabilityReady   RolloutState = "capability_ready"
	RolloutEnabled           RolloutState = "enabled"
	RolloutMigrationPaused   RolloutState = "migration_paused"
	RolloutDegraded          RolloutState = "degraded"
	RolloutCapabilityFailed  RolloutState = "capability_failed"
)

type ContextStrategy string

const (
	StrategyFullHistory      ContextStrategy = "full_history"
	StrategyManagedBootstrap ContextStrategy = "managed_bootstrap"
	StrategyManagedDelta     ContextStrategy = "managed_delta"
)

// ChatUpstreamContextState domain entity.
type ChatUpstreamContextState struct {
	ID                    uint
	ConversationID        uint
	UpstreamID            uint
	UserID                uint
	State                 ContextState
	SessionID             string
	Mode                  string
	Epoch                 int
	EpochHash             string
	BranchLeafMessageID   uint
	LastRequestID         string
	LastBootstrapAt       *time.Time
	LastDeltaAt           *time.Time
	LastErrorCode         string
	CreatedAt             time.Time
	UpdatedAt             time.Time
	UpstreamName          string
	ContextProtocolVersion string
	ContextBuildCommit    string
	ContextBuildTime      string
	ContextBuildVersion   string
	Username              string
}

// ChatContextMetricEvent domain entity.
type ChatContextMetricEvent struct {
	ID               uint
	RequestID        string
	ConversationID   uint
	UpstreamID       uint
	UserID           string
	Username         string
	ContextMode      string
	ProtocolVersion  string
	EpochHash        string
	MessageCount     int
	EstimatedTokens  int64
	SentTokens      int64
	BudgetTokens    int64
	StoredCount      int
	MergedCount      int
	BootstrapCount   int
	DeltaCount       int
	CompactionCount  int
	TrimMessages     int
	TrimTokens       int
	RecallCount      int
	RecallDegraded   bool
	RecallHitsJSON   string
	Status           string
	ErrorCode        string
	LatencyMS        int64
	QueuedAt         int64
	CreatedAt        time.Time
}

// ContextMetricsSummary for overview aggregation.
type ContextMetricsSummary struct {
	ActiveSessions       int64
	PendingMigration     int64
	BootstrapSuccessRate float64
	DeltaSuccessRate    float64
	ContextOverflowCount int64
	P95LatencyMS         int64
	RecallHitCount       int64
	RecallMissCount      int64
	RecallTruncatedCount int64
	EvidenceTokens       int64
}
