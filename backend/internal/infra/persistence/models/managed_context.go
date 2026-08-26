package model

import "time"

// ContextRolloutState values.
const (
	RolloutDisabled          = "disabled"
	RolloutChecking         = "checking"
	RolloutCapabilityReady  = "capability_ready"
	RolloutEnabled          = "enabled"
	RolloutMigrationPaused  = "migration_paused"
	RolloutDegraded         = "degraded"
	RolloutCapabilityFailed = "capability_failed"
)

// ContextStrategy values.
const (
	ContextStrategyFullHistory      = "full_history"
	ContextStrategyManagedBootstrap = "managed_bootstrap"
	ContextStrategyManagedDelta     = "managed_delta"
)

// SessionState values.
const (
	SessionStateLegacy          = "legacy"
	SessionStateBootstrapPending = "bootstrap_pending"
	SessionStateBootstrapping   = "bootstrapping"
	SessionStateDeltaActive      = "delta_active"
	SessionStateDeltaFailed      = "delta_failed"
	SessionStateResyncPending    = "resync_pending"
	SessionStateBootstrapFailed   = "bootstrap_failed"
)

// ChatUpstreamContextState records per-conversation upstream context sync state.
type ChatUpstreamContextState struct {
	ControlPlaneModel
	ConversationID       uint      `gorm:"not null;default:0;index:idx_ctx_state_conv_up;uniqueIndex:idx_ctx_state_unique,priority:1;comment:DEEIX会话ID"`
	UpstreamID           uint      `gorm:"not null;default:0;index:idx_ctx_state_conv_up;uniqueIndex:idx_ctx_state_unique,priority:2;comment:上游ID"`
	UserID               uint      `gorm:"not null;default:0;index:idx_ctx_state_conv_up;uniqueIndex:idx_ctx_state_unique,priority:3;comment:用户ID"`
	SessionID            string    `gorm:"size:64;not null;default:'';index:idx_ctx_state_session;comment:WITHYOU会话ID"`
	State                string    `gorm:"size:32;not null;default:'legacy';index:idx_ctx_state_conv_up;comment:会话状态"`
	Mode                 string    `gorm:"size:32;not null;default:'delta';comment:上下文模式(delta|bootstrap)"`
	Epoch                int       `gorm:"not null;default:0;comment:WITHYOU权威版本号"`
	EpochHash            string    `gorm:"size:128;not null;default:'';comment:截断哈希或不可逆哈希"`
	BranchLeafMessageID  uint      `gorm:"not null;default:0;comment:当前活跃分支叶消息ID"`
	LastRequestID        string    `gorm:"size:64;not null;default:'';comment:最近一次关联请求ID"`
	LastBootstrapAt      *time.Time `gorm:"comment:最近Bootstrap时间"`
	LastDeltaAt          *time.Time `gorm:"comment:最近Delta时间"`
	LastErrorCode        string    `gorm:"size:128;not null;default:'';comment:最近错误码"`
}

func (ChatUpstreamContextState) TableName() string {
	return "chat_upstream_context_states"
}

// ChatContextMetricEvent records context metric events without PII.
type ChatContextMetricEvent struct {
	BaseModel
	RequestID         string `gorm:"size:64;not null;default:'';index:idx_mc_metric_req;comment:请求ID"`
	ConversationID    uint   `gorm:"not null;default:0;index:idx_mc_metric_conv;comment:会话ID"`
	UpstreamID        uint   `gorm:"not null;default:0;index:idx_mc_metric_up;comment:上游ID"`
	UserID            string `gorm:"size:64;not null;default:'';index:idx_mc_metric_user;comment:用户ID"`
	ContextMode       string `gorm:"size:32;not null;default:'';comment:上下文模式"`
	ProtocolVersion   string `gorm:"size:64;not null;default:'';comment:协议版本"`
	EpochHash         string `gorm:"size:128;not null;default:'';comment:截断哈希"`
	MessageCount      int   `gorm:"not null;default:0;comment:消息计数"`
	EstimatedTokens   int64 `gorm:"not null;default:0;comment:估算Token数"`
	SentTokens        int64 `gorm:"not null;default:0;comment:实际发送Token数"`
	BudgetTokens      int64 `gorm:"not null;default:0;comment:上下文预算Token数"`
	StoredCount       int   `gorm:"not null;default:0;comment:存储历史消息数"`
	MergedCount       int   `gorm:"not null;default:0;comment:合并后消息数"`
	BootstrapCount    int   `gorm:"not null;default:0;comment:Bootstrap次数"`
	DeltaCount        int   `gorm:"not null;default:0;comment:Delta次数"`
	CompactionCount   int   `gorm:"not null;default:0;comment:压缩次数"`
	TrimMessages      int   `gorm:"not null;default:0;comment:裁剪消息数"`
	TrimTokens        int   `gorm:"not null;default:0;comment:裁剪Token数"`
	RecallCount       int   `gorm:"not null;default:0;comment:历史召回次数"`
	RecallDegraded    bool  `gorm:"not null;default:false;comment:记忆召回降级"`
	RecallHitsJSON    string `gorm:"type:text;not null;default:'{}';comment:召回命中类别JSON(JSON对象,如{\"brain\":true,\"dialogue\":true})"`
	QueuedAt          int64 `gorm:"not null;default:0;index:idx_mc_metric_up_q;comment:WITHYOU入队时间戳(ms)去重锚点"`
	Status            string `gorm:"size:32;not null;default:'';comment:状态"`
	ErrorCode         string `gorm:"size:128;not null;default:'';comment:错误码"`
	LatencyMS         int64  `gorm:"not null;default:0;comment:延迟毫秒"`
}

func (ChatContextMetricEvent) TableName() string {
	return "chat_context_metric_events"
}
