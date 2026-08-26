package wechatbot

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	appwechatbot "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/wechatbot"
	domain "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/wechatbot"
	domainconversation "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/conversation"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/response"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *appwechatbot.Service
}

func NewHandler(service *appwechatbot.Service) *Handler {
	return &Handler{service: service}
}

type BotQRCodeResponse struct {
	QRCode string `json:"qrcode"`
	QRImg  string `json:"qrcode_img"`
}

type BotStatusResponse struct {
	Online              bool   `json:"online"`
	Status              string `json:"status"`
	ConversationID      uint   `json:"conversation_id,omitempty"`
	ConversationPublicID string `json:"conversation_public_id,omitempty"`
	ExpiresAt           string `json:"expires_at,omitempty"`
}

// requireWeChatAccess 校验当前用户是否有权限使用微信机器人，无权限时返回 403。
func (h *Handler) requireWeChatAccess(c *gin.Context) bool {
	userID := middleware.MustUserID(c)
	allowed, err := h.service.CanUseWeChatBot(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "check wechat access failed")
		return false
	}
	if !allowed {
		response.Error(c, http.StatusForbidden, "wechat bot access denied")
		return false
	}
	return true
}

func (h *Handler) GetQRCode(c *gin.Context) {
	if !h.requireWeChatAccess(c) {
		return
	}
	userID := middleware.MustUserID(c)
	qrcode, qrImg, err := h.service.FetchLoginQRCode(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, appwechatbot.ErrAlreadyBound) {
			response.ErrorWithCode(c, http.StatusConflict, response.CodeWeChatBotAlreadyBound, "already bound")
			return
		}
		response.Error(c, http.StatusInternalServerError, "fetch qrcode failed")
		return
	}
	response.Success(c, BotQRCodeResponse{QRCode: qrcode, QRImg: qrImg})
}

func (h *Handler) PollQRCode(c *gin.Context) {
	if !h.requireWeChatAccess(c) {
		return
	}
	userID := middleware.MustUserID(c)
	status, qrURL, err := h.service.PollQRCodeStatus(c.Request.Context(), fmt.Sprint(userID))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "poll qrcode failed")
		return
	}
	respMap := gin.H{"status": status}
	if qrURL != "" {
		respMap["qrcode_img"] = qrURL
	}
	response.Success(c, respMap)
}

func (h *Handler) GetStatus(c *gin.Context) {
	if !h.requireWeChatAccess(c) {
		return
	}
	userID := middleware.MustUserID(c)
	account, err := h.service.GetBotStatus(c.Request.Context(), userID)
	if errors.Is(err, appwechatbot.ErrBotNotFound) {
		response.Success(c, BotStatusResponse{Online: false})
		return
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "get status failed")
		return
	}
	resp := BotStatusResponse{
		Online:         account.Status == "online",
		Status:         string(account.Status),
		ConversationID: account.ConversationID,
	}
	if account.ExpiresAt != nil {
		resp.ExpiresAt = account.ExpiresAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if account.ConversationID != 0 {
		resp.ConversationPublicID = h.service.GetBotConversationPublicID(c.Request.Context(), userID)
	}
	response.Success(c, resp)
}

func (h *Handler) DeleteBot(c *gin.Context) {
	if !h.requireWeChatAccess(c) {
		return
	}
	userID := middleware.MustUserID(c)
	if err := h.service.DestroyUserBot(c.Request.Context(), userID); err != nil {
		response.Error(c, http.StatusInternalServerError, "delete bot failed")
		return
	}
	response.Success(c, gin.H{"status": "deleted"})
}

func (h *Handler) GetUserBotContacts(c *gin.Context) {
	if !h.requireWeChatAccess(c) {
		return
	}
	userID := middleware.MustUserID(c)
	items, err := h.service.UserListContacts(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "list contacts failed")
		return
	}
	type contactItem struct {
		Wxid              string `json:"wxid"`
		ConversationID    uint   `json:"conversation_id"`
		ConversationTitle string `json:"conversation_title,omitempty"`
		MessageCount      int    `json:"message_count"`
		LastActive        string `json:"last_active"`
	}
	out := make([]contactItem, 0, len(items))
	for _, it := range items {
		var lastActive string
		if !it.SwitchedAt.IsZero() {
			lastActive = it.SwitchedAt.Format(time.RFC3339)
		}
		out = append(out, contactItem{
			Wxid:              it.Wxid,
			ConversationID:    it.ConversationID,
			ConversationTitle: it.ConversationTitle,
			MessageCount:      it.MessageCount,
			LastActive:        lastActive,
		})
	}
	response.Success(c, out)
}

func (h *Handler) GetUserBotAudit(c *gin.Context) {
	if !h.requireWeChatAccess(c) {
		return
	}
	userID := middleware.MustUserID(c)
	logs, err := h.service.UserGetAudit(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "list audit failed")
		return
	}
	type auditItem struct {
		ID         uint      `json:"id"`
		Action     string    `json:"action"`
		DetailJSON string    `json:"detail_json"`
		CreatedAt  time.Time `json:"created_at"`
	}
	items := make([]auditItem, 0, len(logs))
	for _, l := range logs {
		items = append(items, auditItem{
			ID:         l.ID,
			Action:     l.Action,
			DetailJSON: l.DetailJSON,
			CreatedAt:  l.CreatedAt,
		})
	}
	response.Success(c, items)
}

// ---- Admin handlers ----

type AdminBotResponse struct {
	ID                  uint   `json:"id"`
	UserID              uint   `json:"user_id"`
	Username            string `json:"username"`
	UserDisplayName     string `json:"user_display_name"`
	Nickname            string `json:"nickname"`
	WeChatUserID        string `json:"wechat_user_id"`
	Model               string `json:"model"`
	BotModel            string `json:"bot_model"`
	Status              string `json:"status"`
	HubBotID            string `json:"hub_bot_id"`
	ConversationID      uint   `json:"conversation_id"`
	ConversationPublicID string `json:"conversation_public_id,omitempty"`
	ExpiresAt           string `json:"expires_at,omitempty"`
	LastStatusChangeAt  string `json:"last_status_change_at,omitempty"`
	CreatedAt           string `json:"created_at"`
}

func toAdminBotResponse(b domain.BotAccountWithUser) AdminBotResponse {
	r := AdminBotResponse{
		ID:                  b.ID,
		UserID:              b.UserID,
		Username:            b.Username,
		UserDisplayName:     b.UserDisplayName,
		Nickname:            b.Nickname,
		WeChatUserID:        b.WeChatUserID,
		Model:               b.ConversationModel,
		BotModel:            b.BotModel,
		Status:              string(b.Status),
		HubBotID:            b.HubBotID,
		ConversationID:      b.ConversationID,
		ConversationPublicID: b.ConversationPublicID,
		CreatedAt:           b.CreatedAt.Format(time.RFC3339),
	}
	if b.ExpiresAt != nil {
		r.ExpiresAt = b.ExpiresAt.Format(time.RFC3339)
	}
	if b.LastStatusChangeAt != nil {
		r.LastStatusChangeAt = b.LastStatusChangeAt.Format(time.RFC3339)
	}
	return r
}

func (h *Handler) AdminListBots(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	items, total, err := h.service.AdminListBots(c.Request.Context(), page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "list bots failed")
		return
	}
	views := make([]AdminBotResponse, 0, len(items))
	for _, v := range items {
		views = append(views, toAdminBotResponse(v))
	}
	response.Success(c, gin.H{
		"total":         total,
		"results":       views,
		"default_model": h.service.GetBotDefaultModel(),
	})
}

func (h *Handler) AdminDeleteBot(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid user_id")
		return
	}
	if err := h.service.AdminDeleteBot(c.Request.Context(), uint(userID)); err != nil {
		response.Error(c, http.StatusInternalServerError, "delete bot failed")
		return
	}
	response.Success(c, gin.H{"status": "deleted"})
}

type AdminBotDetailResponse struct {
	ID                  uint   `json:"id"`
	UserID              uint   `json:"user_id"`
	Username            string `json:"username"`
	UserDisplayName     string `json:"user_display_name"`
	Nickname            string `json:"nickname"`
	WeChatUserID        string `json:"wechat_user_id"`
	Model               string `json:"model"`
	BotModel            string `json:"bot_model"`
	DefaultModel        string `json:"default_model"`
	Status              string `json:"status"`
	HubBotID            string `json:"hub_bot_id"`
	ConversationID      uint   `json:"conversation_id"`
	ConversationPublicID string `json:"conversation_public_id,omitempty"`
	ExpiresAt           string `json:"expires_at,omitempty"`
	LastStatusChangeAt  string `json:"last_status_change_at,omitempty"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at"`
}

func toAdminBotDetailResponse(r *appwechatbot.AdminBotDetailResult) AdminBotDetailResponse {
	b := r.Bot
	resp := AdminBotDetailResponse{
		ID:                  b.ID,
		UserID:              b.UserID,
		Username:            b.Username,
		UserDisplayName:     b.UserDisplayName,
		Nickname:            b.Nickname,
		WeChatUserID:        b.WeChatUserID,
		Model:               b.ConversationModel,
		BotModel:            b.BotModel,
		DefaultModel:        r.DefaultModel,
		Status:              string(b.Status),
		HubBotID:            b.HubBotID,
		ConversationID:      b.ConversationID,
		ConversationPublicID: r.ConversationPublicID,
		CreatedAt:           b.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           b.UpdatedAt.Format(time.RFC3339),
	}
	if b.ExpiresAt != nil {
		resp.ExpiresAt = b.ExpiresAt.Format(time.RFC3339)
	}
	if b.LastStatusChangeAt != nil {
		resp.LastStatusChangeAt = b.LastStatusChangeAt.Format(time.RFC3339)
	}
	return resp
}

func (h *Handler) AdminGetBot(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid user_id")
		return
	}
	detail, err := h.service.AdminGetBotDetail(c.Request.Context(), uint(userID))
	if err != nil {
		response.Error(c, http.StatusNotFound, "bot not found")
		return
	}
	response.Success(c, toAdminBotDetailResponse(detail))
}

type AdminBotMessageResponse struct {
	ID             uint   `json:"id"`
	Role           string `json:"role"`
	Content        string `json:"content"`
	CreatedAt      string `json:"created_at"`
}

func toAdminBotMessageResponse(m domainconversation.Message) AdminBotMessageResponse {
	return AdminBotMessageResponse{
		ID:        m.ID,
		Role:      m.Role,
		Content:   m.Content,
		CreatedAt: m.CreatedAt.Format(time.RFC3339),
	}
}

func (h *Handler) AdminGetBotMessages(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid user_id")
		return
	}
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}
	messages, total, err := h.service.AdminGetBotMessages(c.Request.Context(), uint(userID), limit)
	if err != nil {
		response.Error(c, http.StatusNotFound, "bot not found")
		return
	}
	views := make([]AdminBotMessageResponse, 0, len(messages))
	for _, m := range messages {
		views = append(views, toAdminBotMessageResponse(m))
	}
	response.SuccessPage(c, total, views)
}

// ---- P2 后台三合一：联系人会话 / 模型配置 / 切换 / 审计 ----

func (h *Handler) AdminGetBotContacts(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid user_id")
		return
	}
	items, err := h.service.AdminListBotContacts(c.Request.Context(), uint(userID))
	if err != nil {
		response.Error(c, http.StatusNotFound, "bot not found")
		return
	}
	response.Success(c, gin.H{"results": items})
}

type adminUpdateModelRequest struct {
	Model string `json:"model"`
}

func (h *Handler) AdminUpdateBotModel(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid user_id")
		return
	}
	var req adminUpdateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	operator := middleware.MustUserID(c)
	err = h.service.AdminUpdateBotModel(
		c.Request.Context(), operator, uint(userID), req.Model,
		c.GetHeader("X-Request-ID"), c.ClientIP(), c.Request.UserAgent(),
	)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "update bot model failed")
		return
	}
	response.Success(c, gin.H{"status": "updated", "model": req.Model})
}

type adminSwitchContactRequest struct {
	Wxid               string `json:"wxid" binding:"required"`
	TargetConversation uint   `json:"target_conversation_id"` // 0 = 新建干净会话
}

// SwitchUserBotContact 用户 scope：切换自己 bot 的联系人会话。
func (h *Handler) SwitchUserBotContact(c *gin.Context) {
	if !h.requireWeChatAccess(c) {
		return
	}
	userID := middleware.MustUserID(c)
	var req adminSwitchContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	targetConvID, err := h.service.UserSwitchContactConversation(
		c.Request.Context(), userID, req.Wxid, req.TargetConversation,
		c.GetHeader("X-Request-ID"), c.ClientIP(), c.Request.UserAgent(),
	)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"status": "switched", "conversation_id": targetConvID})
}

func (h *Handler) AdminSwitchContactConversation(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid user_id")
		return
	}
	var req adminSwitchContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	operator := middleware.MustUserID(c)
	targetConvID, err := h.service.AdminSwitchContactConversation(
		c.Request.Context(), operator, uint(userID), req.Wxid, req.TargetConversation,
		c.GetHeader("X-Request-ID"), c.ClientIP(), c.Request.UserAgent(),
	)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"status": "switched", "conversation_id": targetConvID})
}

func (h *Handler) AdminGetBotAudit(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid user_id")
		return
	}
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}
	logs, total, err := h.service.AdminGetBotAudit(c.Request.Context(), uint(userID), limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "list audit failed")
		return
	}
	type auditItem struct {
		ID          uint      `json:"id"`
		Action      string    `json:"action"`
		ResourceID  string    `json:"resource_id"`
		ActorUserID uint      `json:"actor_user_id"`
		DetailJSON  string    `json:"detail_json"`
		CreatedAt   time.Time `json:"created_at"`
	}
	items := make([]auditItem, 0, len(logs))
	for _, l := range logs {
		items = append(items, auditItem{
			ID:          l.ID,
			Action:      l.Action,
			ResourceID:  l.ResourceID,
			ActorUserID: l.ActorUserID,
			DetailJSON:  l.DetailJSON,
			CreatedAt:   l.CreatedAt,
		})
	}
	response.SuccessPage(c, total, items)
}
