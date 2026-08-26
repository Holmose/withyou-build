package managedcontext

import (
	"net/http"
	"strconv"
	"strings"

	appmc "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/managedcontext"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/response"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

// Handler for managed context admin endpoints.
type Handler struct {
	service *appmc.Service
}

func NewHandler(service *appmc.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetOverview(c *gin.Context) {
	data, err := h.service.GetOverview(c.Request.Context(), nil)
	if err != nil {
		response.ErrorFrom(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) GetMetrics(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	var upstreamID *uint
	if idStr := c.Query("upstreamID"); idStr != "" {
		if id, err := strconv.ParseUint(idStr, 10, 64); err == nil {
			v := uint(id)
			upstreamID = &v
		}
	}

	var conversationID *uint
	if idStr := c.Query("conversationID"); idStr != "" {
		if id, err := strconv.ParseUint(idStr, 10, 64); err == nil {
			v := uint(id)
			conversationID = &v
		}
	}

	rows, total, err := h.service.GetMetrics(c.Request.Context(), upstreamID, conversationID, nil, nil, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessPage(c, total, rows)
}

func (h *Handler) GetRealtime(c *gin.Context) {
	id, err := strconv.ParseUint(c.Query("upstreamID"), 10, 64)
	if err != nil || id == 0 {
		response.ErrorFrom(c, http.StatusBadRequest, err)
		return
	}
	data, err := h.service.GetRealtimeStatus(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorFrom(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}

func (h *Handler) LookupMetricEvent(c *gin.Context) {
	upstreamID, err := strconv.ParseUint(c.Query("upstreamID"), 10, 64)
	requestID := strings.TrimSpace(c.Query("requestID"))
	if err != nil || upstreamID == 0 || requestID == "" {
		response.ErrorFrom(c, http.StatusBadRequest, nil)
		return
	}
	row, err := h.service.GetMetricEventByRequest(c.Request.Context(), uint(upstreamID), requestID)
	if err != nil {
		response.ErrorFrom(c, http.StatusInternalServerError, err)
		return
	}
	if row == nil {
		response.Success(c, nil)
		return
	}
	response.Success(c, row)
}

func (h *Handler) ListSessions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	var upstreamID *uint
	if idStr := c.Query("upstreamID"); idStr != "" {
		if id, err := strconv.ParseUint(idStr, 10, 64); err == nil {
			v := uint(id)
			upstreamID = &v
		}
	}

	var state *string
	if s := c.Query("state"); s != "" {
		state = &s
	}

	rows, total, err := h.service.ListSessions(c.Request.Context(), upstreamID, state, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessPage(c, total, rows)
}

func (h *Handler) CheckUpstream(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid upstream id")
		return
	}
	actorUserID := middleware.MustUserID(c)
	requestID := middleware.MustRequestID(c)

	result, err := h.service.CheckCapabilities(c.Request.Context(), uint(id), actorUserID, requestID, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		response.ErrorFrom(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) TestUpstream(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid upstream id")
		return
	}
	actorUserID := middleware.MustUserID(c)
	requestID := middleware.MustRequestID(c)

	result, err := h.service.TestProtocol(c.Request.Context(), uint(id), actorUserID, requestID, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		response.ErrorFrom(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) EnableUpstream(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid upstream id")
		return
	}
	actorUserID := middleware.MustUserID(c)
	requestID := middleware.MustRequestID(c)

	err = h.service.EnableUpstream(c.Request.Context(), uint(id), actorUserID, requestID, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, gin.H{"enabled": true})
}

func (h *Handler) PauseUpstream(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid upstream id")
		return
	}
	actorUserID := middleware.MustUserID(c)
	requestID := middleware.MustRequestID(c)

	err = h.service.PauseUpstream(c.Request.Context(), uint(id), actorUserID, requestID, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, gin.H{"paused": true})
}

func (h *Handler) DisableUpstream(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid upstream id")
		return
	}
	actorUserID := middleware.MustUserID(c)
	requestID := middleware.MustRequestID(c)

	err = h.service.DisableUpstream(c.Request.Context(), uint(id), actorUserID, requestID, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, gin.H{"disabled": true})
}

func (h *Handler) ResyncConversation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid conversation id")
		return
	}
	actorUserID := middleware.MustUserID(c)
	requestID := middleware.MustRequestID(c)

	err = h.service.ResyncConversation(c.Request.Context(), uint(id), actorUserID, requestID, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		response.ErrorFrom(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, gin.H{"resynced": true})
}
