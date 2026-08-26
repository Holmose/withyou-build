package llmpool

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/channel"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
)

func trimBaseURL(url string) string {
	url = strings.TrimRight(url, "/")
	if strings.HasSuffix(url, "/v1") {
		return url[:len(url)-3]
	}
	return url
}

func TestTrimBaseURL(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"http://host/v1", "http://host"},
		{"http://host/v1/", "http://host"},
		{"http://host/api/v2", "http://host/api/v2"},
		{"http://host", "http://host"},
		{"http://host/", "http://host"},
		{"http://host/v1/beta", "http://host/v1/beta"},
		{"http://host/abc/v1", "http://host/abc"},
		{"http://host/abc/v1/", "http://host/abc"},
	}
	for _, tt := range tests {
		got := trimBaseURL(tt.input)
		if got != tt.want {
			t.Errorf("trim(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

type fakeChanRepo struct {
	repository.ChannelRepository
	rows []repository.ChannelUpstreamListRow
}

func (f *fakeChanRepo) ListUpstreams(context.Context, repository.ListChannelUpstreamsInput) ([]repository.ChannelUpstreamListRow, int64, error) {
	return f.rows, int64(len(f.rows)), nil
}

func TestPoolHealthCacheSingleFlight(t *testing.T) {
	var hits atomic.Int64
	seenPath := make(chan string, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		seenPath <- r.URL.Path
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"providers":[{"name":"p1"}]}`))
	}))
	defer srv.Close()

	repo := &fakeChanRepo{rows: []repository.ChannelUpstreamListRow{{
		Upstream: channel.Upstream{Name: "withyou-test", BaseURL: srv.URL + "/v1", Status: "active"},
	}}}
	s := NewService(repo, nil)

	const callers = 5
	var wg sync.WaitGroup
	results := make([]*PoolHealthResult, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = s.PoolHealth(context.Background())
		}(i)
	}
	wg.Wait()

	if got := hits.Load(); got != 1 {
		t.Fatalf("upstream hit %d times, want 1 (single-flight)", got)
	}
	if p := <-seenPath; p != "/health/providers" {
		t.Fatalf("probe path = %q, want /health/providers (v1 stripped)", p)
	}
	for i, r := range results {
		if !r.Online || len(r.Sources) != 1 || len(r.Sources[0].Providers) != 1 {
			t.Fatalf("result[%d] unexpected: %+v", i, r)
		}
	}

	if again := s.PoolHealth(context.Background()); again != results[0] {
		t.Fatal("cached call should return the shared result within TTL")
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("cache miss after %v, upstream hit again (%d)", cacheTTL, got)
	}
}
