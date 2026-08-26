package llmpool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"go.uber.org/zap"
)

// withYouNamePrefix 命中 llm_upstreams 中 WithYou 池实例的名称前缀。
const withYouNamePrefix = "withyou"

const probeTimeout = 35 * time.Second

// cacheTTL 池心跳缓存时长。WithYou /health/providers 现场探测全模型需 ~24s，
// 前端 10s 轮询会请求堆积；持锁缓存使探测期间进来的轮询共享同一结果。
const cacheTTL = 30 * time.Second

// Source 一个 WithYou 实例的聚合结果；providers 为 WithYou /health/providers 原样透传。
type Source struct {
	Upstream   string            `json:"upstream"`
	BaseURL    string            `json:"base_url"`
	Online     bool              `json:"online"`
	Error      string            `json:"error,omitempty"`
	Providers  []json.RawMessage `json:"providers,omitempty"`
}

// PoolHealthResult GET /admin/llm-pool/health 响应负载。
type PoolHealthResult struct {
	Online  bool     `json:"online"`
	Sources []Source `json:"sources"`
}

// Service 探测 WithYou 模型池心跳。
type Service struct {
	repo repository.ChannelRepository
	http *http.Client
	log  *zap.Logger

	cacheMu sync.Mutex
	cache   *PoolHealthResult
	cacheAt time.Time
}

// NewService 创建服务。
func NewService(repo repository.ChannelRepository, log *zap.Logger) *Service {
	return &Service{
		repo: repo,
		http: &http.Client{Timeout: probeTimeout},
		log:  log,
	}
}

// PoolHealth 并发探测所有 WithYou 上游实例的 /health/providers。
// 全部不可达时 Online=false；单实例失败不阻塞其它实例。
// 结果缓存 cacheTTL；探测期间并发轮询在锁上排队，共享同一次探测结果（single-flight）。
func (s *Service) PoolHealth(ctx context.Context) *PoolHealthResult {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if s.cache != nil && time.Since(s.cacheAt) < cacheTTL {
		return s.cache
	}
	result := s.probeAll(ctx)
	s.cache = result
	s.cacheAt = time.Now()
	return result
}

func (s *Service) probeAll(ctx context.Context) *PoolHealthResult {
	result := &PoolHealthResult{Sources: []Source{}}
	if s.repo == nil {
		return result
	}
	rows, _, err := s.repo.ListUpstreams(ctx, repository.ListChannelUpstreamsInput{
		Query:  withYouNamePrefix,
		Status: "active",
		Limit:  100,
	})
	if err != nil {
		s.logWarn("llm_pool_list_upstreams_failed", err)
		return result
	}

	var targets []struct{ name, baseURL string }
	for _, row := range rows {
		name, baseURL := strings.ToLower(row.Upstream.Name), strings.ToLower(row.Upstream.BaseURL)
		if !strings.Contains(name, withYouNamePrefix) && !strings.Contains(baseURL, withYouNamePrefix) {
			continue
		}
		if strings.TrimSpace(row.Upstream.BaseURL) == "" {
			continue
		}
		baseURL = strings.TrimRight(row.Upstream.BaseURL, "/")
		if strings.HasSuffix(baseURL, "/v1") {
			baseURL = baseURL[:len(baseURL)-3]
		}
		targets = append(targets, struct{ name, baseURL string }{row.Upstream.Name, baseURL})
	}
	if len(targets) == 0 {
		return result
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, t := range targets {
		wg.Add(1)
		go func(name, baseURL string) {
			defer wg.Done()
			src := s.probe(ctx, name, baseURL)
			mu.Lock()
			result.Sources = append(result.Sources, src)
			if src.Online {
				result.Online = true
			}
			mu.Unlock()
		}(t.name, t.baseURL)
	}
	wg.Wait()
	return result
}

func (s *Service) probe(ctx context.Context, name, baseURL string) Source {
	src := Source{Upstream: name, BaseURL: baseURL}
	reqCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, baseURL+"/health/providers", nil)
	if err != nil {
		src.Error = "build request failed"
		return src
	}
	resp, err := s.http.Do(req)
	if err != nil {
		src.Error = "unreachable"
		s.logWarn("llm_pool_probe_failed", err)
		return src
	}
	defer resp.Body.Close() //nolint:errcheck
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		src.Error = fmt.Sprintf("status %d", resp.StatusCode)
		return src
	}
	var payload struct {
		Providers []json.RawMessage `json:"providers"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.Providers == nil {
		// 兼容顶层直接是数组的情况
		var arr []json.RawMessage
		if json.Unmarshal(body, &arr) == nil {
			payload.Providers = arr
		} else {
			src.Error = "invalid payload"
			return src
		}
	}
	src.Online = true
	src.Providers = payload.Providers
	return src
}

func (s *Service) logWarn(msg string, err error) {
	if s.log != nil {
		s.log.Warn(msg, zap.Error(err))
	}
}
