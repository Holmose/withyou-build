package llm

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetAdditionalHeadersResolvesAllowListedIdentityTemplates(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "https://example.com", nil)
	err := setAdditionalHeaders(req, `{"X-Tenant-Id":"withyou","X-User-Id":"{{user_id}}","X-Conversation-Id":"{{conversation_id}}"}`, HeaderTemplateValues{UserID: 12, ConversationID: 386})
	if err != nil {
		t.Fatal(err)
	}
	if got := req.Header.Get("X-Tenant-Id"); got != "withyou" {
		t.Fatalf("expected tenant header, got %q", got)
	}
	if got := req.Header.Get("X-User-Id"); got != "12" {
		t.Fatalf("expected user header, got %q", got)
	}
	if got := req.Header.Get("X-Conversation-Id"); got != "386" {
		t.Fatalf("expected conversation header, got %q", got)
	}
}

func TestSetAdditionalHeadersKeepsStaticHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "https://example.com", nil)
	if err := setAdditionalHeaders(req, `{"X-Title":"WithYou","X-Retry":3}`, HeaderTemplateValues{}); err != nil {
		t.Fatal(err)
	}
	if got := req.Header.Get("X-Title"); got != "WithYou" {
		t.Fatalf("expected static title, got %q", got)
	}
	if got := req.Header.Get("X-Retry"); got != "3" {
		t.Fatalf("expected stringified number, got %q", got)
	}
}

func TestSetAdditionalHeadersUsesModelListProbeIdentity(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://example.com/v1/models", nil)
	err := setAdditionalHeaders(req, `{"X-Tenant-Id":"withyou","X-User-Id":"{{user_id}}","X-Conversation-Id":"{{conversation_id}}"}`, modelListHeaderTemplateValues())
	if err != nil {
		t.Fatal(err)
	}
	if got := req.Header.Get("X-User-Id"); got != "1" {
		t.Fatalf("expected probe user header, got %q", got)
	}
	if got := req.Header.Get("X-Conversation-Id"); got != "1" {
		t.Fatalf("expected probe conversation header, got %q", got)
	}
}

func TestFetchOpenAIResponseUsesProbeIdentityForDynamicHeaders(t *testing.T) {
	var gotUser, gotConversation string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = r.Header.Get("X-User-Id")
		gotConversation = r.Header.Get("X-Conversation-Id")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_1","output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}`))
	}))
	defer server.Close()

	output, err := NewClient().RetrieveOpenAIResponse(t.Context(), RouteConfig{
		Protocol:    AdapterOpenAIResponses,
		BaseURL:     server.URL,
		HeadersJSON: `{"X-User-Id":"{{user_id}}","X-Conversation-Id":"{{conversation_id}}"}`,
	}, "resp_1")
	if err != nil {
		t.Fatal(err)
	}
	if output.Text != "ok" {
		t.Fatalf("expected output text, got %#v", output)
	}
	if gotUser != "1" || gotConversation != "1" {
		t.Fatalf("expected probe headers, got user=%q conversation=%q", gotUser, gotConversation)
	}
}

func TestSetAdditionalHeadersRejectsUnsafeTemplates(t *testing.T) {
	cases := []struct {
		name   string
		raw    string
		values HeaderTemplateValues
	}{
		{name: "missing user", raw: `{"X-User-Id":"{{user_id}}"}`},
		{name: "unknown variable", raw: `{"X-Account":"{{account_id}}"}`, values: HeaderTemplateValues{UserID: 1, ConversationID: 2}},
		{name: "template in key", raw: `{"X-{{user_id}}":"value"}`, values: HeaderTemplateValues{UserID: 1, ConversationID: 2}},
		{name: "invalid json", raw: `{`, values: HeaderTemplateValues{UserID: 1, ConversationID: 2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "https://example.com", nil)
			if err := setAdditionalHeaders(req, tc.raw, tc.values); err == nil {
				t.Fatal("expected local configuration error")
			}
		})
	}
}
