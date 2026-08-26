package managedcontext

import (
	"context"
	"strings"
	"time"

	domain "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/managedcontext"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/persistence/dberror"
	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/persistence/models"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"gorm.io/gorm"
)

func translateError(err error) error {
	if dberror.IsRecordNotFound(err) {
		return repository.ErrNotFound
	}
	return err
}

type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) UpsertContextState(ctx context.Context, item *domain.ChatUpstreamContextState) error {
	m := toModelContextState(item)
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND upstream_id = ? AND user_id = ?", item.ConversationID, item.UpstreamID, item.UserID).
		Assign(model.ChatUpstreamContextState{
			SessionID:            m.SessionID,
			Mode:                 m.Mode,
			Epoch:                m.Epoch,
			State:                m.State,
			EpochHash:            m.EpochHash,
			BranchLeafMessageID: m.BranchLeafMessageID,
			LastRequestID:        m.LastRequestID,
			LastBootstrapAt:      m.LastBootstrapAt,
			LastDeltaAt:          m.LastDeltaAt,
			LastErrorCode:        m.LastErrorCode,
		}).FirstOrCreate(&m).Error
	if err != nil {
		return translateError(err)
	}
	item.ID = m.ID
	item.CreatedAt = m.CreatedAt
	item.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *Repo) GetContextStateByConversationID(ctx context.Context, conversationID uint) (*domain.ChatUpstreamContextState, error) {
	var m model.ChatUpstreamContextState
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("id DESC").
		First(&m).Error
	if err != nil {
		return nil, translateError(err)
	}
	res := toDomainContextState(&m)
	return &res, nil
}

func (r *Repo) GetContextState(ctx context.Context, conversationID, upstreamID, userID uint) (*domain.ChatUpstreamContextState, error) {
	var m model.ChatUpstreamContextState
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND upstream_id = ? AND user_id = ?", conversationID, upstreamID, userID).
		First(&m).Error
	if err != nil {
		return nil, translateError(err)
	}
	res := toDomainContextState(&m)
	return &res, nil
}

func (r *Repo) ListContextStates(ctx context.Context, offset, limit int, upstreamID *uint, userID *uint, state *string) ([]domain.ChatUpstreamContextState, int64, error) {
	var total int64
	countQuery := r.db.WithContext(ctx).Model(&model.ChatUpstreamContextState{})
	if upstreamID != nil {
		countQuery = countQuery.Where("upstream_id = ?", *upstreamID)
	}
	if userID != nil {
		countQuery = countQuery.Where("user_id = ?", *userID)
	}
	if state != nil && *state != "" {
		countQuery = countQuery.Where("state = ?", *state)
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, translateError(err)
	}

	type stateWithUpstream struct {
		model.ChatUpstreamContextState
		UpstreamName           string
		ContextProtocolVersion string
		ContextBuildCommit     string
		ContextBuildTime       string
		ContextBuildVersion    string
		Username              string
	}

	var items []stateWithUpstream
	query := r.db.WithContext(ctx).Table("chat_upstream_context_states cs").
		Select("cs.*, u.name as upstream_name, u.context_protocol_version, u.context_build_commit, u.context_build_time, u.context_build_version, us.username").
		Joins("LEFT JOIN llm_upstreams u ON cs.upstream_id = u.id").
		Joins("LEFT JOIN identity_users us ON CASE WHEN cs.user_id::text ~ '^[0-9]+$' THEN us.id = cs.user_id ELSE false END").
		Order("cs.id DESC").Offset(offset).Limit(limit)
	if upstreamID != nil {
		query = query.Where("cs.upstream_id = ?", *upstreamID)
	}
	if userID != nil {
		query = query.Where("cs.user_id = ?", *userID)
	}
	if state != nil && *state != "" {
		query = query.Where("cs.state = ?", *state)
	}
	if err := query.Find(&items).Error; err != nil {
		return nil, 0, translateError(err)
	}

	results := make([]domain.ChatUpstreamContextState, 0, len(items))
	for _, it := range items {
		ds := toDomainContextState(&it.ChatUpstreamContextState)
		ds.UpstreamName = it.UpstreamName
		ds.ContextProtocolVersion = it.ContextProtocolVersion
		ds.ContextBuildCommit = it.ContextBuildCommit
		ds.ContextBuildTime = it.ContextBuildTime
		ds.ContextBuildVersion = it.ContextBuildVersion
		ds.Username = it.Username
		results = append(results, ds)
	}
	return results, total, nil
}

func (r *Repo) CountSessionsByState(ctx context.Context, upstreamID uint) (map[string]int64, error) {
	type result struct {
		State string
		Count int64
	}
	var rows []result
	err := r.db.WithContext(ctx).
		Model(&model.ChatUpstreamContextState{}).
		Select("state, COUNT(*) as count").
		Where("upstream_id = ?", upstreamID).
		Group("state").
		Find(&rows).Error
	if err != nil {
		return nil, translateError(err)
	}
	m := make(map[string]int64, len(rows))
	for _, r := range rows {
		m[r.State] = r.Count
	}
	return m, nil
}

func (r *Repo) CreateMetricEvent(ctx context.Context, item *domain.ChatContextMetricEvent) error {
	m := toModelMetricEvent(item)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return translateError(err)
	}
	item.ID = m.ID
	item.CreatedAt = m.CreatedAt
	return nil
}

func (r *Repo) MetricEventExists(ctx context.Context, upstreamID uint, requestID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ChatContextMetricEvent{}).
		Where("upstream_id = ? AND request_id = ? AND request_id != ''", upstreamID, requestID).
		Count(&count).Error
	if err != nil {
		return false, translateError(err)
	}
	return count > 0, nil
}

// MetricEventExistsByQueuedAt checks idempotency on the WITHYOU queued-at
// timestamp instead of request_id. WITHYOU's request tracer resets its
// per-process id counter on restart (llm-1, llm-2, ...), so request_id alone
// is not unique across restarts and would silently drop fresh metrics.
func (r *Repo) MetricEventExistsByQueuedAt(ctx context.Context, upstreamID uint, queuedAt int64) (bool, error) {
	if queuedAt <= 0 {
		return false, nil
	}
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ChatContextMetricEvent{}).
		Where("upstream_id = ? AND queued_at = ?", upstreamID, queuedAt).
		Count(&count).Error
	if err != nil {
		return false, translateError(err)
	}
	return count > 0, nil
}

func (r *Repo) ListMetricEvents(ctx context.Context, offset, limit int, upstreamID *uint, conversationID *uint, createdFrom, createdTo *time.Time) ([]domain.ChatContextMetricEvent, int64, error) {
	type metricWithUser struct {
		model.ChatContextMetricEvent
		Username string
	}
	var total int64
	countQuery := r.db.WithContext(ctx).Model(&model.ChatContextMetricEvent{})
	if upstreamID != nil {
		countQuery = countQuery.Where("upstream_id = ?", *upstreamID)
	}
	if conversationID != nil {
		countQuery = countQuery.Where("conversation_id = ?", *conversationID)
	}
	if createdFrom != nil {
		countQuery = countQuery.Where("created_at >= ?", *createdFrom)
	}
	if createdTo != nil {
		countQuery = countQuery.Where("created_at <= ?", *createdTo)
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, translateError(err)
	}

	var items []metricWithUser
	query := r.db.WithContext(ctx).Table("chat_context_metric_events m").
		Select("m.*, u.username").
		Joins("LEFT JOIN identity_users u ON CASE WHEN m.user_id ~ '^[0-9]+$' THEN u.id = m.user_id::bigint ELSE false END").
		Order("m.id DESC").Offset(offset).Limit(limit)
	if upstreamID != nil {
		query = query.Where("m.upstream_id = ?", *upstreamID)
	}
	if conversationID != nil {
		query = query.Where("m.conversation_id = ?", *conversationID)
	}
	if createdFrom != nil {
		query = query.Where("m.created_at >= ?", *createdFrom)
	}
	if createdTo != nil {
		query = query.Where("m.created_at <= ?", *createdTo)
	}
	if err := query.Find(&items).Error; err != nil {
		return nil, 0, translateError(err)
	}

	results := make([]domain.ChatContextMetricEvent, 0, len(items))
	for _, it := range items {
		d := toDomainMetricEvent(&it.ChatContextMetricEvent)
		d.Username = it.Username
		results = append(results, d)
	}
	return results, total, nil
}

// GetLatestMetricEventByRequestID returns the newest event for a tracer
// request id (ids reset on WithYou restart, so "latest" is the useful one).
func (r *Repo) GetLatestMetricEventByRequestID(ctx context.Context, upstreamID uint, requestID string) (*domain.ChatContextMetricEvent, error) {
	type metricWithUser struct {
		model.ChatContextMetricEvent
		Username string
	}
	var items []metricWithUser
	err := r.db.WithContext(ctx).Table("chat_context_metric_events m").
		Select("m.*, u.username").
		Joins("LEFT JOIN identity_users u ON CASE WHEN m.user_id ~ '^[0-9]+$' THEN u.id = m.user_id::bigint ELSE false END").
		Where("m.upstream_id = ? AND m.request_id = ?", upstreamID, requestID).
		Order("m.id DESC").Limit(1).
		Find(&items).Error
	if err != nil {
		return nil, translateError(err)
	}
	if len(items) == 0 {
		return nil, nil
	}
	d := toDomainMetricEvent(&items[0].ChatContextMetricEvent)
	d.Username = items[0].Username
	return &d, nil
}

func (r *Repo) MetricsSummary(ctx context.Context, upstreamID *uint, from, to *time.Time) (*domain.ContextMetricsSummary, error) {
	type countResult struct {
		Total int64 `gorm:"column:total"`
	}
	type sumResult struct {
		SumLatencyMS int64 `gorm:"column:sum_latency"`
		SumEvidence  int64 `gorm:"column:sum_evidence"`
		TotalEvents  int64 `gorm:"column:total_events"`
	}

	s := &domain.ContextMetricsSummary{}

	q := r.db.WithContext(ctx).Model(&model.ChatContextMetricEvent{})
	if upstreamID != nil {
		q = q.Where("upstream_id = ?", *upstreamID)
	}
	if from != nil {
		q = q.Where("created_at >= ?", *from)
	}
	if to != nil {
		q = q.Where("created_at <= ?", *to)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, translateError(err)
	}
	if total == 0 {
		return s, nil
	}

	{
		var p95 float64
		rawSQL := `SELECT COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms), 0) FROM chat_context_metric_events WHERE deleted_at IS NULL`
		args := []interface{}{}
		if upstreamID != nil {
			rawSQL += ` AND upstream_id = ?`
			args = append(args, *upstreamID)
		}
		if from != nil {
			rawSQL += ` AND created_at >= ?`
			args = append(args, *from)
		}
		if to != nil {
			rawSQL += ` AND created_at <= ?`
			args = append(args, *to)
		}
		if err := r.db.WithContext(ctx).Raw(rawSQL, args...).Scan(&p95).Error; err != nil {
			return nil, translateError(err)
		}
		s.P95LatencyMS = int64(p95)
	}

	var recallHit int64
	_ = r.db.WithContext(ctx).Model(&model.ChatContextMetricEvent{}).Select("COALESCE(SUM(recall_count),0) as recall_sum").Scan(&recallHit)

	sq := r.db.WithContext(ctx).Model(&model.ChatUpstreamContextState{})
	if upstreamID != nil {
		sq = sq.Where("upstream_id = ?", *upstreamID)
	}
	sq.Count(&s.ActiveSessions)

	var pending int64
	r.db.WithContext(ctx).Model(&model.ChatUpstreamContextState{}).
		Where("upstream_id = ? AND state IN ?", upstreamID, []string{"bootstrap_pending", "resync_pending"}).
		Count(&pending)
	s.PendingMigration = pending

	return s, nil
}

func (r *Repo) ListUpstreamsWithContextInfo(ctx context.Context, offset, limit int, stateFilter string) ([]repository.ManagedContextUpstreamRow, int64, error) {
	type row struct {
		ID                             uint
		Name                           string
		BaseURL                        string
		ContextStrategy                string
		ContextProtocolVersion         string
		ContextBuildCommit             string
		ContextBuildTime               string
		ContextBuildVersion            string
		ManagedContextRolloutState     string
		ContextCapabilitiesJSON        string
		ContextCapabilitiesCheckedAt   *time.Time
		ContextCapabilitiesError       string
	}
	var items []row
	var total int64

	query := r.db.WithContext(ctx).Table("llm_upstreams").
		Select("id, name, base_url, context_strategy, context_protocol_version, context_build_commit, context_build_time, context_build_version, managed_context_rollout_state, context_capabilities_json, context_capabilities_checked_at, context_capabilities_error").
		Where("context_strategy IS NOT NULL AND context_strategy != ''")

	if stateFilter != "" {
		query = query.Where("managed_context_rollout_state = ?", stateFilter)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, translateError(err)
	}
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Scan(&items).Error; err != nil {
		return nil, 0, translateError(err)
	}

	results := make([]repository.ManagedContextUpstreamRow, 0, len(items))
	for _, it := range items {
		var activeSessions int64
		r.db.WithContext(ctx).Model(&model.ChatUpstreamContextState{}).
			Where("upstream_id = ? AND state IN ?", it.ID, []string{"delta_active", "bootstrapping"}).
			Count(&activeSessions)
		var pending int64
		r.db.WithContext(ctx).Model(&model.ChatUpstreamContextState{}).
			Where("upstream_id = ? AND state IN ?", it.ID, []string{"bootstrap_pending", "resync_pending"}).
			Count(&pending)

		var cb *repository.ContextBuild
		if it.ContextBuildCommit != "" || it.ContextBuildTime != "" || it.ContextBuildVersion != "" {
			cb = &repository.ContextBuild{
				Commit:  it.ContextBuildCommit,
				Time:    it.ContextBuildTime,
				Version: it.ContextBuildVersion,
			}
		}

		results = append(results, repository.ManagedContextUpstreamRow{
			ID:                              it.ID,
			Name:                            it.Name,
			BaseURL:                         it.BaseURL,
			ContextStrategy:                 domain.ContextStrategy(it.ContextStrategy),
			ContextProtocolVersion:          it.ContextProtocolVersion,
			ContextBuild:                    cb,
			ManagedContextRolloutState:      domain.RolloutState(it.ManagedContextRolloutState),
			ContextCapabilitiesJSON:         it.ContextCapabilitiesJSON,
			ContextCapabilitiesCheckedAt:    it.ContextCapabilitiesCheckedAt,
			ContextCapabilitiesError:        it.ContextCapabilitiesError,
			ActiveSessions:                  activeSessions,
			PendingMigration:                pending,
		})
	}
	return results, total, nil
}

func toModelContextState(d *domain.ChatUpstreamContextState) model.ChatUpstreamContextState {
	return model.ChatUpstreamContextState{
		ControlPlaneModel: model.ControlPlaneModel{
			ID:        d.ID,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		},
		ConversationID:      d.ConversationID,
		UpstreamID:          d.UpstreamID,
		UserID:              d.UserID,
		SessionID:           d.SessionID,
		State:               string(d.State),
		Mode:                d.Mode,
		Epoch:               d.Epoch,
		EpochHash:           d.EpochHash,
		BranchLeafMessageID: d.BranchLeafMessageID,
		LastRequestID:       d.LastRequestID,
		LastBootstrapAt:     d.LastBootstrapAt,
		LastDeltaAt:         d.LastDeltaAt,
		LastErrorCode:       d.LastErrorCode,
	}
}

func toDomainContextState(m *model.ChatUpstreamContextState) domain.ChatUpstreamContextState {
	return domain.ChatUpstreamContextState{
		ID:               m.ID,
		ConversationID:   m.ConversationID,
		UpstreamID:       m.UpstreamID,
		UserID:           m.UserID,
		SessionID:        strings.TrimSpace(m.SessionID),
		State:            domain.ContextState(m.State),
		Mode:             m.Mode,
		Epoch:            m.Epoch,
		EpochHash:        strings.TrimSpace(m.EpochHash),
		BranchLeafMessageID: m.BranchLeafMessageID,
		LastRequestID:    strings.TrimSpace(m.LastRequestID),
		LastBootstrapAt:  m.LastBootstrapAt,
		LastDeltaAt:      m.LastDeltaAt,
		LastErrorCode:    strings.TrimSpace(m.LastErrorCode),
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

func toModelMetricEvent(d *domain.ChatContextMetricEvent) model.ChatContextMetricEvent {
	return model.ChatContextMetricEvent{
		BaseModel: model.BaseModel{
			ID:        d.ID,
			CreatedAt: d.CreatedAt,
		},
		RequestID:        d.RequestID,
		ConversationID:   d.ConversationID,
		UpstreamID:       d.UpstreamID,
		UserID:           d.UserID,
		ContextMode:      d.ContextMode,
		ProtocolVersion:  d.ProtocolVersion,
		EpochHash:        d.EpochHash,
		MessageCount:     d.MessageCount,
		EstimatedTokens:  d.EstimatedTokens,
		SentTokens:       d.SentTokens,
		BudgetTokens:     d.BudgetTokens,
		StoredCount:      d.StoredCount,
		MergedCount:      d.MergedCount,
		BootstrapCount:   d.BootstrapCount,
		DeltaCount:       d.DeltaCount,
		CompactionCount:  d.CompactionCount,
		TrimMessages:     d.TrimMessages,
		TrimTokens:       d.TrimTokens,
		RecallCount:      d.RecallCount,
		RecallDegraded:   d.RecallDegraded,
		RecallHitsJSON:   d.RecallHitsJSON,
		QueuedAt:         d.QueuedAt,
		Status:           d.Status,
		ErrorCode:        d.ErrorCode,
		LatencyMS:        d.LatencyMS,
	}
}

func toDomainMetricEvent(m *model.ChatContextMetricEvent) domain.ChatContextMetricEvent {
	return domain.ChatContextMetricEvent{
		ID:              m.ID,
		RequestID:       m.RequestID,
		ConversationID: m.ConversationID,
		UpstreamID:     m.UpstreamID,
		UserID:         m.UserID,
		ContextMode:    m.ContextMode,
		ProtocolVersion: m.ProtocolVersion,
		EpochHash:      m.EpochHash,
		MessageCount:   m.MessageCount,
		EstimatedTokens: m.EstimatedTokens,
		SentTokens:     m.SentTokens,
		BudgetTokens:   m.BudgetTokens,
		StoredCount:    m.StoredCount,
		MergedCount:    m.MergedCount,
		BootstrapCount: m.BootstrapCount,
		DeltaCount:     m.DeltaCount,
		CompactionCount: m.CompactionCount,
		TrimMessages:   m.TrimMessages,
		TrimTokens:     m.TrimTokens,
		RecallCount:    m.RecallCount,
		RecallDegraded: m.RecallDegraded,
		RecallHitsJSON: m.RecallHitsJSON,
		QueuedAt:       m.QueuedAt,
		Status:         m.Status,
		ErrorCode:      m.ErrorCode,
		LatencyMS:      m.LatencyMS,
		CreatedAt:      m.CreatedAt,
	}
}
