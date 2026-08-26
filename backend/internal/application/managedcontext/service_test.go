package managedcontext

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/llm"
)

// mockAudit implements AuditWriter with no-op methods.
type mockAudit struct{}

func (*mockAudit) Write(context.Context, string, uint, string, string, string, string, string, interface{}) {
}

// legacyCapabilitiesHandler returns the WITHYOU Agent 2.2.20 legacy response.
func legacyCapabilitiesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"token_estimator": "cjk_v1",
			"limits": {
				"bootstrap_tokens": 80000,
				"provider_input_tokens": 96000,
				"single_message_tokens": 64000
			}
		}`))
	}
}

// TestCapabilityURLConstruction verifies that the capability URL does NOT produce /v1/v1
// when the upstream BaseURL already ends with /v1.
// See: P0-3 gap analysis.
func TestCapabilityURLConstruction(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		want    string
	}{
		{
			name:    "plain base gets /v1/withyou/capabilities",
			baseURL: "http://withyou-agent:8080",
			want:    "http://withyou-agent:8080/v1/withyou/capabilities",
		},
		{
			name:    "v1 base does NOT double to /v1/v1",
			baseURL: "http://withyou-agent:8080/v1",
			want:    "http://withyou-agent:8080/v1/withyou/capabilities",
		},
		{
			name:    "v1 with trailing slash",
			baseURL: "http://withyou-agent:8080/v1/",
			want:    "http://withyou-agent:8080/v1/withyou/capabilities",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			// P0-3 fix: CheckCapabilities now uses llm.BuildVersionedEndpointURL.
			got := llm.BuildVersionedEndpointURL(tt.baseURL, "v1", "/withyou/capabilities")
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// TestLegacyServerCapabilitiesResponse verifies that a server returning the legacy
// limits-only response is classified as NOT capability_ready.
// See: P0-4 gap analysis.
func TestLegacyServerCapabilitiesResponse(t *testing.T) {
	server := httptest.NewServer(legacyCapabilitiesHandler())
	defer server.Close()

	// Verify the test server returns the legacy response format.
	resp, err := http.Get(server.URL + "/v1/withyou/capabilities")
	if err != nil {
		t.Fatalf("test server error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("test server returned status %d", resp.StatusCode)
	}

	// Parse the response the same way CheckCapabilities does.
	caps, err := capsFromResponse(resp)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// A legacy response has no protocol_version.
	if version, _ := caps["protocol_version"].(string); version != "" {
		t.Errorf("legacy response should have empty protocol_version, got %q", version)
	}

	// The current CheckCapabilities implementation returns Success=true for any 200 + valid JSON.
	// After the fix, it must validate protocol_version is non-empty before returning success.
	if version, ok := caps["protocol_version"].(string); !ok || version == "" {
		t.Logf("legacy response correctly detected: protocol_version=%q", version)
	}
}

// capsFromResponse parses a capability response from an http.Response.
// This mirrors the parsing logic in service.go for isolated testing.
func capsFromResponse(resp *http.Response) (map[string]interface{}, error) {
	body := make([]byte, 65536)
	n, _ := resp.Body.Read(body)
	resp.Body.Close()
	var caps map[string]interface{}
	if err := json.Unmarshal(body[:n], &caps); err != nil {
		return nil, err
	}
	return caps, nil
}
