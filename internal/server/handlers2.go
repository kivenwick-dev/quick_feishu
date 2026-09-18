package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"quick-feishu/internal/app"
	"quick-feishu/internal/config"
	"quick-feishu/internal/db"
	"quick-feishu/internal/feishu"
	"quick-feishu/internal/model"
	"quick-feishu/internal/report"
)

func (h *Handlers) ListSnapshots(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	list, total, err := db.ListSnapshots(h.DB, page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败，请检查输入或服务端配置"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": publicSnapshots(list), "total": total})
}

func (h *Handlers) GetSnapshot(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var snap model.Snapshot
	if err := h.DB.First(&snap, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"snapshot": publicSnapshot(&snap)})
}

func (h *Handlers) CompareSnapshots(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	if from == "" || to == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from and to required"})
		return
	}
	early, earlyErr := db.SnapshotByDate(h.DB, from)
	late, lateErr := db.SnapshotByDate(h.DB, to)
	if earlyErr != nil || lateErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"from": gin.H{"date": early.SnapshotDate, "used": early.AccountUsed, "quota": early.AccountQuota, "requests": early.RequestCount},
		"to":   gin.H{"date": late.SnapshotDate, "used": late.AccountUsed, "quota": late.AccountQuota, "requests": late.RequestCount},
		"diff": gin.H{"used": late.AccountUsed - early.AccountUsed, "requests": late.RequestCount - early.RequestCount},
	})
}

func (h *Handlers) GetTemplate(c *gin.Context) {
	c.JSON(http.StatusOK, h.Config.ReportTemplate)
}

func (h *Handlers) SaveTemplate(c *gin.Context) {
	var tmpl report.Template
	if err := c.ShouldBindJSON(&tmpl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "操作失败，请检查输入或服务端配置"})
		return
	}
	if err := h.App.SaveTemplate(mustToMap(tmpl)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败，请检查输入或服务端配置"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) GetDict(c *gin.Context) {
	source := c.Param("source")
	var items []interface{}
	switch source {
	case "account":
		var list []model.DictAccountField
		h.DB.Find(&list)
		for _, v := range list {
			items = append(items, v)
		}
	case "token":
		var list []model.DictTokenField
		h.DB.Find(&list)
		for _, v := range list {
			items = append(items, v)
		}
	case "usage":
		var list []model.DictUsageField
		h.DB.Find(&list)
		for _, v := range list {
			items = append(items, v)
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad source"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// SaveDict 新增或更新某接口的一个字段字典项（按 field_path upsert）
func (h *Handlers) SaveDict(c *gin.Context) {
	source := c.Param("source")
	var in struct {
		FieldPath   string `json:"field_path"`
		Label       string `json:"label"`
		FieldType   string `json:"field_type"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "操作失败，请检查输入或服务端配置"})
		return
	}
	if in.FieldPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "field_path required"})
		return
	}
	switch source {
	case "account":
		var f model.DictAccountField
		if err := h.DB.Where("field_path = ?", in.FieldPath).First(&f).Error; err != nil {
			f = model.DictAccountField{FieldPath: in.FieldPath}
		}
		f.Label, f.FieldType, f.Description = in.Label, in.FieldType, in.Description
		if err := h.DB.Save(&f).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败，请检查输入或服务端配置"})
			return
		}
	case "token":
		var f model.DictTokenField
		if err := h.DB.Where("field_path = ?", in.FieldPath).First(&f).Error; err != nil {
			f = model.DictTokenField{FieldPath: in.FieldPath}
		}
		f.Label, f.FieldType, f.Description = in.Label, in.FieldType, in.Description
		if err := h.DB.Save(&f).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败，请检查输入或服务端配置"})
			return
		}
	case "usage":
		var f model.DictUsageField
		if err := h.DB.Where("field_path = ?", in.FieldPath).First(&f).Error; err != nil {
			f = model.DictUsageField{FieldPath: in.FieldPath}
		}
		f.Label, f.FieldType, f.Description = in.Label, in.FieldType, in.Description
		if err := h.DB.Save(&f).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败，请检查输入或服务端配置"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad source"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) GetSettings(c *gin.Context) {
	public := *h.Config
	public.Account.UserID = ""
	public.Account.SystemToken = ""
	public.Account.APIBase = ""
	public.Feishu.WebhookURL = ""
	c.JSON(http.StatusOK, gin.H{
		"config":         public,
		"configured":     gin.H{"user_id": h.Config.Account.UserID != "", "system_token": h.Config.Account.SystemToken != "", "api_base": h.Config.Account.APIBase != "", "webhook_url": h.Config.Feishu.WebhookURL != ""},
		"env_overridden": config.EnvOverridden(),
	})
}

func (h *Handlers) SaveSettings(c *gin.Context) {
	var in config.Config
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "操作失败，请检查输入或服务端配置"})
		return
	}
	if err := h.App.SaveConfig(&in); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败，请检查输入或服务端配置"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// RestartScheduler 重新加载配置、重建 API 客户端、重启定时任务，并重新采集当天快照，
// 使新账号/令牌立即生效。
func (h *Handlers) RestartScheduler(c *gin.Context) {
	if err := h.App.Restart(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败，请检查输入或服务端配置"})
		return
	}
	res, snapErr := h.App.RunSnapshot()
	resp := gin.H{
		"success":       true,
		"next_runs":     h.App.NextRuns(),
		"snapshot_date": app.Today(),
	}
	if snapErr != nil {
		resp["snapshot_error"] = "采集失败，请检查服务端配置或日志"
	} else if res != nil && len(res.Errors) > 0 {
		resp["snapshot_warnings"] = publicWarnings(res.Errors)
	}
	c.JSON(http.StatusOK, resp)
}

// SchedulerStatus 返回定时任务时间与下次执行时间
func (h *Handlers) SchedulerStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"snapshot_time": h.Config.Schedule.SnapshotTime,
		"report_time":   h.Config.Schedule.ReportTime,
		"next_runs":     h.App.NextRuns(),
	})
}

func (h *Handlers) TestFeishu(c *gin.Context) {
	client := feishu.NewClient(h.Config.Feishu.WebhookURL, 1)
	body := []byte(`{"msg_type":"text","content":{"text":"QuickFeishu 测试消息"}}`)
	_, err := client.SendCard(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "操作失败，请检查输入或服务端配置"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) ListSendLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	list, total, err := db.ListSendLogs(h.DB, page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败，请检查输入或服务端配置"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": publicLogs(list), "total": total})
}

func mustToMap(v interface{}) map[string]interface{} {
	b, _ := json.Marshal(v)
	var m map[string]interface{}
	json.Unmarshal(b, &m)
	return m
}
