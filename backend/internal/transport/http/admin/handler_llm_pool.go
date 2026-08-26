package admin

import (
	"context"
	"net/http"

	appllmpool "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/llmpool"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/response"
	"github.com/gin-gonic/gin"
)

// llmPoolProber 模型池心跳探测能力（application/llmpool）。
type llmPoolProber interface {
	PoolHealth(ctx context.Context) *appllmpool.PoolHealthResult
}

var _ llmPoolProber = (*appllmpool.Service)(nil)

// SetLLMPoolProber 注入模型池心跳探测器。
func (h *Handler) SetLLMPoolProber(prober llmPoolProber) {
	h.llmPoolProber = prober
}

// ProxyLLMPoolHealth godoc
// @Summary 管理员查询 WithYou 模型池心跳
// @Description 代理转发 WithYou /health/providers（按 llm_upstreams 中 withyou* 实例聚合）
// @Tags Admin
// @Produce json
// @Success 200 {object} response.Response
// @Router /admin/llm-pool/health [get]
func (h *Handler) ProxyLLMPoolHealth(c *gin.Context) {
	if h.llmPoolProber == nil {
		response.Error(c, http.StatusNotImplemented, "llm pool prober not configured")
		return
	}
	result := h.llmPoolProber.PoolHealth(c.Request.Context())
	// 始终 200 + online 标记，前端离线态渲染不触发错误弹窗
	response.Success(c, result)
}