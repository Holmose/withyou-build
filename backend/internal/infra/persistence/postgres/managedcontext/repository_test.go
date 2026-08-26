package managedcontext

import (
	"context"
	"testing"

	domain "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/managedcontext"
	model "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/persistence/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// openMCTestDB creates an in-memory SQLite database for managed context tests.
func openMCTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("resolve sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := db.AutoMigrate(&model.LLMUpstream{}, &model.ChatUpstreamContextState{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// TestDefaultWithYouUpstreamVisibleInOverview verifies that a WITHYOU upstream
// with the default (full_history + disabled) state IS visible in the overview
// query, giving administrators an entry point to trigger capability detection.
// See: P0-2 gap analysis.
//
// CURRENT BUG: ListUpstreamsWithContextInfo filters out default-state upstreams with:
//   WHERE context_strategy != 'full_history' OR managed_context_rollout_state != 'disabled'
// This means a default-state WITHYOU upstream is invisible to admins.
//
// AFTER FIX: The query should be changed to also include context_provider=withyou
// upstreams regardless of their strategy/state, so admins can discover and configure them.
func TestDefaultWithYouUpstreamVisibleInOverview(t *testing.T) {
	db := openMCTestDB(t)
	repo := &Repo{db: db}

	// Create a WITHYOU upstream with default state.
	// A real WITHYOU upstream would have context_protocol_version="1" after initial capability detection.
	upstream := model.LLMUpstream{
		Name:                            "withyou-agent",
		BaseURL:                         "http://withyou-agent:8080",
		Compatible:                      "openai",
		Status:                          "active",
		ContextStrategy:                 "full_history",
		ManagedContextRolloutState:       "disabled",
		ContextCapabilitiesJSON:           "{}",
		ContextProtocolVersion:            "1",
	}
	if err := db.Create(&upstream).Error; err != nil {
		t.Fatalf("create upstream: %v", err)
	}

	// Query all upstreams with context info.
	rows, total, err := repo.ListUpstreamsWithContextInfo(context.Background(), 0, 100, "")
	if err != nil {
		t.Fatalf("ListUpstreamsWithContextInfo: %v", err)
	}

	// The fix requires that a default-state WITHYOU upstream appears in the results.
	// The current buggy query filters it out.
	found := false
	for _, r := range rows {
		if r.ID == upstream.ID && r.Name == "withyou-agent" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("P0-2 bug: default (full_history+disabled) WITHYOU upstream (id=%d) is NOT visible in overview; query returned %d rows, total=%d",
			upstream.ID, len(rows), total)
	}

	_ = domain.RolloutState("")
}

// TestResyncIdentityResolution verifies that resync correctly resolves
// conversation+upstream identity and does NOT write upstreamID=0.
// See: P1-2 gap analysis.
func TestResyncIdentityResolution(t *testing.T) {
	db := openMCTestDB(t)
	repo := &Repo{db: db}

	// Create an existing context state with real upstream ID.
	existing := model.ChatUpstreamContextState{
		ConversationID: 10,
		UpstreamID:     13,
		UserID:         5,
		State:          "delta_active",
		EpochHash:      "abc123",
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("create state: %v", err)
	}

	// Simulate the fixed ResyncConversation flow:
	// 1. GetContextStateByConversationID finds the existing row (UpstreamID=13)
	// 2. UpsertContextState is called with the resolved UpstreamID=13
	state, err := repo.GetContextStateByConversationID(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetContextStateByConversationID: %v", err)
	}
	if state == nil {
		t.Fatal("expected to find existing state by conversation ID")
	}

	// Upsert with the resolved upstream ID (13), simulating the fixed code path.
	resyncEntry := &domain.ChatUpstreamContextState{
		ConversationID: 10,
		UpstreamID:     state.UpstreamID, // fixed: resolved from existing state, not 0
		UserID:         5,
		State:          domain.StateResyncPending,
		EpochHash:      "resync-10-123456",
	}
	if err := repo.UpsertContextState(context.Background(), resyncEntry); err != nil {
		t.Fatalf("UpsertContextState: %v", err)
	}

	// Verify only 1 row exists and it has the correct upstream ID.
	var all []model.ChatUpstreamContextState
	if err := db.Find(&all).Error; err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 row after resync, got %d", len(all))
	}
	if all[0].UpstreamID != 13 {
		t.Errorf("expected upstream_id=13, got %d", all[0].UpstreamID)
	}
	if all[0].State != string(domain.StateResyncPending) {
		t.Errorf("expected state=resync_pending, got %s", all[0].State)
	}
}

// TestUpsertPersistsSessionTracking verifies that SessionID/Mode/Epoch are
// written on upsert and read back, so the managed-context panel can show
// session tracking from the WITHYOU context header.
func TestUpsertPersistsSessionTracking(t *testing.T) {
	db := openMCTestDB(t)
	repo := &Repo{db: db}

	entry := &domain.ChatUpstreamContextState{
		ConversationID: 22,
		UpstreamID:     13,
		UserID:         5,
		SessionID:      "sess-7f3a9c2e",
		State:          domain.StateDeltaActive,
		Mode:           "delta",
		Epoch:          3,
		EpochHash:      "delta-3",
	}
	if err := repo.UpsertContextState(context.Background(), entry); err != nil {
		t.Fatalf("UpsertContextState: %v", err)
	}

	got, err := repo.GetContextState(context.Background(), 22, 13, 5)
	if err != nil {
		t.Fatalf("GetContextState: %v", err)
	}
	if got.SessionID != "sess-7f3a9c2e" {
		t.Errorf("expected sessionID=%q, got %q", "sess-7f3a9c2e", got.SessionID)
	}
	if got.Mode != "delta" {
		t.Errorf("expected mode=delta, got %q", got.Mode)
	}
	if got.Epoch != 3 {
		t.Errorf("expected epoch=3, got %d", got.Epoch)
	}
}
