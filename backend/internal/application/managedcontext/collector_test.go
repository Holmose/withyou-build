package managedcontext

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMatchContextMetricByTimestamp(t *testing.T) {
	now := time.Now().UnixMilli()
	metrics := []withyouContextMetricEvent{
		{RequestID: "req-a", CreatedAt: now - 4000, TotalMs: 100},
		{RequestID: "req-b", CreatedAt: now - 1000, TotalMs: 200},
		{RequestID: "req-c", CreatedAt: now + 3000, TotalMs: 300},
	}
	entry := withyouTraceEntry{ID: "llm-1", QueuedAt: now, DurationMs: 195}

	got := matchContextMetric(metrics, entry)
	if got == nil {
		t.Fatal("expected a match")
	}
	if got.RequestID != "req-b" {
		t.Fatalf("expected req-b (nearest createdAt), got %s", got.RequestID)
	}
}

func TestMatchContextMetricOutsideWindow(t *testing.T) {
	now := time.Now().UnixMilli()
	metrics := []withyouContextMetricEvent{
		{RequestID: "req-far", CreatedAt: now - 60_000, TotalMs: 100},
	}
	entry := withyouTraceEntry{ID: "llm-1", QueuedAt: now, DurationMs: 195}
	if got := matchContextMetric(metrics, entry); got != nil {
		t.Fatalf("expected no match outside 10s window, got %s", got.RequestID)
	}
}

func TestInheritContextMetric(t *testing.T) {
	cm := withyouContextMetricEvent{
		Status:          "failed",
		ErrorCode:       "provider_unavailable",
		TotalMs:         1234,
		ContextMode:     "delta",
		ProtocolVersion: 1,
		Merged: &struct {
			MessageCount    int   `json:"messageCount"`
			EstimatedTokens int64 `json:"estimatedTokens"`
		}{MessageCount: 7, EstimatedTokens: 999},
		Final: &struct {
			MessageCount    int   `json:"messageCount"`
			EstimatedTokens int64 `json:"estimatedTokens"`
			BudgetTokens    int64 `json:"budgetTokens"`
		}{MessageCount: 7, EstimatedTokens: 666},
		Provider: &struct {
			Provider             string `json:"provider"`
			Model                string `json:"model"`
			Status               string `json:"status"`
			ErrorCode            string `json:"errorCode"`
			EstimatedInputTokens int64  `json:"estimatedInputTokens"`
			ActualInputTokens    int64  `json:"actualInputTokens"`
			ActualOutputTokens   int64  `json:"actualOutputTokens"`
			DurationMs           int64  `json:"durationMs"`
		}{ActualInputTokens: 2868, Status: "success"},
	}

	var status, errorCode, mode, pv string
	var latency int64
	var msgCount int
	var est, sent int64

	inheritContextMetric(&cm, &status, &errorCode, &latency, &msgCount, &est, &mode, &pv, &sent)

	if status != "failed" || errorCode != "provider_unavailable" {
		t.Fatalf("status/error not inherited: %q/%q", status, errorCode)
	}
	if latency != 1234 {
		t.Fatalf("latency not inherited: %d", latency)
	}
	if mode != "delta" || pv != "1" {
		t.Fatalf("mode/pv not inherited: %q/%q", mode, pv)
	}
	if msgCount != 7 || est != 666 {
		t.Fatalf("msg/est not inherited: %d/%d", msgCount, est)
	}
	if sent != 2868 {
		t.Fatalf("sent tokens not inherited: %d", sent)
	}
}

func TestInheritContextDetail(t *testing.T) {
	cm := withyouContextMetricEvent{
		Stored: &struct {
			MessageCount    int   `json:"messageCount"`
			EstimatedTokens int64 `json:"estimatedTokens"`
		}{MessageCount: 3, EstimatedTokens: 841},
		Merged: &struct {
			MessageCount    int   `json:"messageCount"`
			EstimatedTokens int64 `json:"estimatedTokens"`
		}{MessageCount: 4, EstimatedTokens: 849},
		Final: &struct {
			MessageCount    int   `json:"messageCount"`
			EstimatedTokens int64 `json:"estimatedTokens"`
			BudgetTokens    int64 `json:"budgetTokens"`
		}{MessageCount: 4, EstimatedTokens: 8894, BudgetTokens: 96000},
		Recall: &struct {
			Executed bool            `json:"executed"`
			HitCount int             `json:"hitCount"`
			Degraded bool            `json:"degraded"`
			Hits     json.RawMessage `json:"hits"`
		}{Executed: true, HitCount: 2, Degraded: false,
			Hits: json.RawMessage(`{"brain":true,"dialogue":true}`)},
	}

	var budget int64
	var stored, merged, recall int
	var degraded bool
	var recallHits *withyouRecallHits
	var trimMessages, trimTokens int
	var epochHash string
	inheritContextDetail(&cm, &budget, &stored, &merged, &recall, &degraded, &recallHits, &trimMessages, &trimTokens, &epochHash)

	if budget != 96000 || stored != 3 || merged != 4 {
		t.Fatalf("budget/stored/merged not inherited: %d/%d/%d", budget, stored, merged)
	}
	if recall != 2 || degraded {
		t.Fatalf("recall not inherited: count=%d degraded=%v", recall, degraded)
	}
	if recallHits == nil {
		t.Fatal("recall hits should be inherited")
	}

	cm2 := withyouContextMetricEvent{}
	budget, stored, merged, recall = 0, 0, 0, 0
	degraded = false
	recallHits = nil
	trimMessages, trimTokens = 0, 0
	epochHash = ""
	inheritContextDetail(&cm2, &budget, &stored, &merged, &recall, &degraded, &recallHits, &trimMessages, &trimTokens, &epochHash)
	if budget != 0 || stored != 0 || merged != 0 || recall != 0 || degraded {
		t.Fatalf("empty metric should leave fields zero: %d/%d/%d/%d/%v", budget, stored, merged, recall, degraded)
	}
	if recallHits != nil {
		t.Fatal("empty metric should leave recall hits nil")
	}
}

func TestUnmarshalCompactionAndRecallHits(t *testing.T) {
	raw := `{
	  "requestId": "llm-1",
	  "contextMode": "delta",
	  "status": "success",
	  "totalMs": 100,
	  "protocolVersion": 1,
	  "createdAt": 1730000000000,
	  "stored": {"messageCount": 120, "estimatedTokens": 200000},
	  "merged": {"messageCount": 34, "estimatedTokens": 34000},
	  "final": {"messageCount": 8, "estimatedTokens": 9000, "budgetTokens": 96000},
	  "provider": {"provider": "minimax", "model": "MiniMax-M2.7", "status": "success",
	    "estimatedInputTokens": 9000, "actualInputTokens": 6232, "actualOutputTokens": 120},
	  "recall": {"executed": true, "hitCount": 3,
	    "hits": {"brain": false, "userProfile": true, "student": true, "dialogue": true, "counterpart": false}},
	  "compaction": {"count": 1}
	}`

	var cm withyouContextMetricEvent
	if err := json.Unmarshal([]byte(raw), &cm); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if cm.Compaction == nil || cm.Compaction.Count != 1 {
		t.Fatalf("compaction not parsed: %+v", cm.Compaction)
	}
	if cm.Recall == nil || len(cm.Recall.Hits) == 0 {
		t.Fatalf("recall hits not parsed: %+v", cm.Recall)
	}
	if hits := parseRecallHits(cm.Recall.Hits); hits == nil || (*hits)["dialogue"].Count != 1 {
		t.Fatalf("recall hits not normalized: %+v", cm.Recall.Hits)
	}
	var status, errorCode, mode, pv string
	var latency int64
	var msgCount int
	var est, sent int64
	inheritContextMetric(&cm, &status, &errorCode, &latency, &msgCount, &est, &mode, &pv, &sent)
	if sent != 6232 {
		t.Fatalf("sent tokens not inherited: %d", sent)
	}
	if est != 9000 {
		t.Fatalf("final est tokens not inherited: %d", est)
	}
}

func TestParseUserIDFromSessionArray(t *testing.T) {
	cases := []struct {
		session string
		want    string
	}{
		{`["withyou-session-v1","withyou","101","1296"]`, "101"},

		{"", ""},
	}
	for _, c := range cases {
		if got := parseUserID(c.session); got != c.want {
			t.Fatalf("parseUserID(%q) = %q, want %q", c.session, got, c.want)
		}
	}
}

func TestUnquoteValue(t *testing.T) {
	cases := []struct{ in, want string }{
		{`"101"`, "101"},
		{`101`, "101"},
		{` "101" `, "101"},
	}
	for _, c := range cases {
		if got := unquoteValue(c.in); got != c.want {
			t.Fatalf("unquoteValue(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
