package server

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:web
var webFS embed.FS

func (s *Server) RegisterRoutes(h *Handlers) {
	api := s.Engine.Group("/api")
	{
		api.GET("/dashboard", h.Dashboard)
		api.POST("/snapshot/run", h.RunSnapshot)
		api.POST("/report/send", h.SendReport)
		api.GET("/snapshots", h.ListSnapshots)
		api.GET("/compare", h.CompareSnapshots)
		api.GET("/snapshots/:id", h.GetSnapshot)
		api.GET("/template", h.GetTemplate)
		api.PUT("/template", h.SaveTemplate)
		api.GET("/dict/:source", h.GetDict)
		api.PUT("/dict/:source", h.SaveDict)
		api.GET("/settings", h.GetSettings)
		api.PUT("/settings", h.SaveSettings)
		api.POST("/feishu/test", h.TestFeishu)
		api.GET("/sendlogs", h.ListSendLogs)
	}
	staticFS, _ := fs.Sub(webFS, "web")
	fileServer := http.FileServer(http.FS(staticFS))
	// gin 的根 catch-all 路由与 /api 静态前缀冲突（radix tree 限制），
	// 故用 NoRoute 回退托管嵌入的前端 SPA。
	s.Engine.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}
