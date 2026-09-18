package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": list, "total": total})
}

func (h *Handlers) GetSnapshot(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var snap model.Snapshot
	if err := h.DB.First(&snap, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	tokens, _ := db.TokenSnapshots(h.DB, snap.ID)
	c.JSON(http.StatusOK, gin.H{"snapshot": snap, "tokens": tokens})
}

func (h *Handlers) CompareSnapshots(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	early, _ := db.SnapshotByDate(h.DB, from)
	late, _ := db.SnapshotByDate(h.DB, to)
	if early == nil || late == nil {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.Config.ReportTemplate = mustToMap(tmpl)
	if err := h.Config.Save(h.ConfigPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	case "token":
		var f model.DictTokenField
		if err := h.DB.Where("field_path = ?", in.FieldPath).First(&f).Error; err != nil {
			f = model.DictTokenField{FieldPath: in.FieldPath}
		}
		f.Label, f.FieldType, f.Description = in.Label, in.FieldType, in.Description
		if err := h.DB.Save(&f).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	case "usage":
		var f model.DictUsageField
		if err := h.DB.Where("field_path = ?", in.FieldPath).First(&f).Error; err != nil {
			f = model.DictUsageField{FieldPath: in.FieldPath}
		}
		f.Label, f.FieldType, f.Description = in.Label, in.FieldType, in.Description
		if err := h.DB.Save(&f).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad source"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) GetSettings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"config":         h.Config,
		"env_overridden": config.EnvOverridden(),
	})
}

func (h *Handlers) SaveSettings(c *gin.Context) {
	var in config.Config
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	env := config.EnvOverridden()
	if env["user_id"] {
		in.Account.UserID = h.Config.Account.UserID
	}
	if env["system_token"] {
		in.Account.SystemToken = h.Config.Account.SystemToken
	}
	if env["webhook_url"] {
		in.Feishu.WebhookURL = h.Config.Feishu.WebhookURL
	}
	*h.Config = in
	if err := h.Config.Save(h.ConfigPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handlers) TestFeishu(c *gin.Context) {
	client := feishu.NewClient(h.Config.Feishu.WebhookURL, 1)
	body := []byte(`{"msg_type":"text","content":{"text":"QuickFeishu 测试消息"}}`)
	resp, err := client.SendCard(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "resp": resp})
}

func (h *Handlers) ListSendLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	list, total, err := db.ListSendLogs(h.DB, page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": list, "total": total})
}

func mustToMap(v interface{}) map[string]interface{} {
	b, _ := json.Marshal(v)
	var m map[string]interface{}
	json.Unmarshal(b, &m)
	return m
}
