package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"quick-feishu/internal/app"
	"quick-feishu/internal/db"
	"quick-feishu/internal/model"
)

type Handlers struct {
	App *app.App
}

func (h *Handlers) Dashboard(c *gin.Context) {
	gdb := h.App.DB()
	snap, err := db.LatestSnapshot(gdb)
	if err != nil {
		snap = nil
	}
	var logs []model.SendLog
	gdb.Order("id DESC").Limit(10).Find(&logs)
	c.JSON(http.StatusOK, gin.H{"latest_snapshot": publicSnapshot(snap), "recent_logs": publicLogs(logs)})
}

func (h *Handlers) RunSnapshot(c *gin.Context) {
	res, err := h.App.RunSnapshot()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败，请检查输入或服务端配置"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "date": app.Today(), "issues": publicIssues(res.Issues)})
}

func (h *Handlers) SendReport(c *gin.Context) {
	log, err := h.App.RunReport()
	if log == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "操作失败，请检查输入或服务端配置"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败，请检查输入或服务端配置", "log_id": log.ID})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "log_id": log.ID})
}
