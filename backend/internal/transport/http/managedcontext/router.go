package managedcontext

import "github.com/gin-gonic/gin"

func (m *Module) RegisterRoutes(adminGroup *gin.RouterGroup) {
	adminGroup.GET("/managed-context/overview", m.Handler.GetOverview)
	adminGroup.GET("/managed-context/realtime", m.Handler.GetRealtime)
	adminGroup.GET("/managed-context/metrics/lookup", m.Handler.LookupMetricEvent)
	adminGroup.GET("/managed-context/metrics", m.Handler.GetMetrics)
	adminGroup.GET("/managed-context/sessions", m.Handler.ListSessions)
	adminGroup.POST("/managed-context/upstreams/:id/check", m.Handler.CheckUpstream)
	adminGroup.POST("/managed-context/upstreams/:id/test", m.Handler.TestUpstream)
	adminGroup.POST("/managed-context/upstreams/:id/enable", m.Handler.EnableUpstream)
	adminGroup.POST("/managed-context/upstreams/:id/pause", m.Handler.PauseUpstream)
	adminGroup.POST("/managed-context/upstreams/:id/disable", m.Handler.DisableUpstream)
	adminGroup.POST("/managed-context/conversations/:id/resync", m.Handler.ResyncConversation)
}
