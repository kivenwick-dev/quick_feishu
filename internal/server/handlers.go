package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"quick-feishu/internal/app"
	"quick-feishu/internal/config"
	"quick-feishu/internal/db"
	"quick-feishu/internal/model"
)

type Handlers struct {
	App        *app.App
	DB         *gorm.DB
	Config     *config.Config
	ConfigPath string
}

func (h *Handlers) Dashboard(c *gin.Context) {
	snap, err := db.LatestSnapshot(h.DB)
	if err != nil {
		snap = nil
	}
	var logs []model.SendLog
	h.DB.Order("id DESC").Limit(10).Find(&logs)
	c.JSON(http.StatusOK, gin.H{"latest_snapshot": snap, "recent_logs": logs})
}

func (h *Handlers) RunSnapshot(c *gin.Context) {
	res, err := h.App.RunSnapshot()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "date": app.Today(), "errors": res.Errors})
}

func (h *Handlers) SendReport(c *gin.Context) {
	log, err := h.App.RunReport()
	if log == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "log_id": log.ID})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "log_id": log.ID})
}
