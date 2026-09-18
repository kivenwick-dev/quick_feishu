package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"quick-feishu/internal/api"
	"quick-feishu/internal/collector"
	"quick-feishu/internal/config"
	"quick-feishu/internal/db"
	"quick-feishu/internal/model"
	"quick-feishu/internal/report"
)

type Handlers struct {
	DB         *gorm.DB
	Config     *config.Config
	ConfigPath string
	APIClient  *api.Client
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
	res := collector.Collect(h.APIClient)
	today := timeNow()
	if err := collector.Save(h.DB, today, res); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "date": today, "errors": res.Errors})
}

func (h *Handlers) SendReport(c *gin.Context) {
	latest, _ := db.LatestSnapshot(h.DB)
	if latest == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no snapshots yet, run snapshot first"})
		return
	}
	prev, _ := db.SnapshotBefore(h.DB, addDays(latest.SnapshotDate, -1))
	tmpl := report.DefaultTemplate()
	log, err := report.ExecuteReport(h.DB, latest, prev, tmpl, h.Config.Feishu.WebhookURL, h.Config.Feishu.RetryTimes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "log_id": log.ID})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "log_id": log.ID})
}
