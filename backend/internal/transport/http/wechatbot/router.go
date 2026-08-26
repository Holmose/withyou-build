package wechatbot

import "github.com/gin-gonic/gin"

type Module struct {
	Handler *Handler
}

func NewModule(handler *Handler) *Module {
	return &Module{Handler: handler}
}

func (m *Module) RegisterRoutes(authRequired *gin.RouterGroup) {
	authRequired.GET("/wechat/bot/qrcode", m.Handler.GetQRCode)
	authRequired.GET("/wechat/bot/qrcode/poll", m.Handler.PollQRCode)
	authRequired.GET("/wechat/bot/status", m.Handler.GetStatus)
	authRequired.DELETE("/wechat/bot", m.Handler.DeleteBot)
	authRequired.GET("/wechat/bot/contacts", m.Handler.GetUserBotContacts)
	authRequired.POST("/wechat/bot/contacts/switch", m.Handler.SwitchUserBotContact)
	authRequired.GET("/wechat/bot/audit", m.Handler.GetUserBotAudit)
}

func (m *Module) RegisterAdminRoutes(adminGroup *gin.RouterGroup) {
	adminGroup.GET("/wechat-bots", m.Handler.AdminListBots)
	adminGroup.GET("/wechat-bots/:user_id", m.Handler.AdminGetBot)
	adminGroup.GET("/wechat-bots/:user_id/messages", m.Handler.AdminGetBotMessages)
	adminGroup.GET("/wechat-bots/:user_id/contacts", m.Handler.AdminGetBotContacts)
	adminGroup.GET("/wechat-bots/:user_id/audit", m.Handler.AdminGetBotAudit)
	adminGroup.PATCH("/wechat-bots/:user_id/model", m.Handler.AdminUpdateBotModel)
	adminGroup.POST("/wechat-bots/:user_id/contacts/switch", m.Handler.AdminSwitchContactConversation)
	adminGroup.DELETE("/wechat-bots/:user_id", m.Handler.AdminDeleteBot)
}
