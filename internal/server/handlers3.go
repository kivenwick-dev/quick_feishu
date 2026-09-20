package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"quick-feishu/internal/db"
	"quick-feishu/internal/model"
	"quick-feishu/internal/report"
)

// GetLiveBilling reads the current account balance and platform conversion rate directly.
// It deliberately bypasses snapshots so the dashboard can refresh monetary values in real time.
func (h *Handlers) GetLiveBilling(c *gin.Context) {
	client := h.App.Client()
	account, _, accountErr := client.GetAccount()
	status, statusErr := client.GetStatus()
	if accountErr != nil || statusErr != nil || account == nil || status == nil || status.QuotaPerUnit <= 0 {
		c.JSON(http.StatusBadGateway, gin.H{"error": "无法获取实时账务数据"})
		return
	}
	rate := status.QuotaPerUnit
	c.JSON(http.StatusOK, gin.H{
		"quota":               account.Quota,
		"used_quota":          account.UsedQuota,
		"quota_per_unit":      rate,
		"balance_usd":         float64(account.Quota) / float64(rate),
		"used_usd":            float64(account.UsedQuota) / float64(rate),
		"currency":            "USD",
		"display_in_currency": status.DisplayInCurrency,
		"updated_at":          time.Now(),
	})
}

// GetQuotaRate returns the platform's live quota-to-currency conversion setting.
func (h *Handlers) GetQuotaRate(c *gin.Context) {
	status, err := h.App.Client().GetStatus()
	if err != nil || status.QuotaPerUnit <= 0 {
		c.JSON(http.StatusBadGateway, gin.H{"error": "无法获取额度换算配置"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"quota_per_unit":      status.QuotaPerUnit,
		"display_in_currency": status.DisplayInCurrency,
		"currency":            "USD",
	})
}

// GetHistory 返回某接口的每日指标矩阵（账号信息 / 令牌使用情况）
func (h *Handlers) GetHistory(c *gin.Context) {
	source := c.DefaultQuery("source", "account")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	switch source {
	case "account":
		c.JSON(http.StatusOK, publicHistory(report.BuildAccountHistory(h.App.DB(), limit)))
	case "usage":
		tokenID, _ := strconv.Atoi(c.Query("token_id"))
		if tokenID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "token_id required"})
			return
		}
		c.JSON(http.StatusOK, publicHistory(report.BuildUsageHistory(h.App.DB(), tokenID, limit)))
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad source"})
	}
}

// GetTokens 返回最近快照下的令牌列表（供历史页选择令牌）
func (h *Handlers) GetTokens(c *gin.Context) {
	toks, err := db.LatestTokens(h.App.DB())
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"items": []interface{}{}})
		return
	}
	type item struct {
		TokenID int    `json:"token_id"`
		Name    string `json:"token_name"`
	}
	items := []item{}
	for _, t := range toks {
		items = append(items, item{TokenID: t.TokenID, Name: t.TokenName})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// GetLatest 返回最近快照按当前模板组装的分区结果（含变动），供仪表盘预览
func (h *Handlers) GetLatest(c *gin.Context) {
	gdb := h.App.DB()
	latest, err := db.LatestSnapshot(gdb)
	if err != nil || latest == nil {
		c.JSON(http.StatusOK, gin.H{"date": nil, "sections": []interface{}{}})
		return
	}
	var prev *model.Snapshot
	if p, e := db.PreviousSnapshot(gdb, latest); e == nil {
		prev = p
	}
	tmpl, _ := report.TemplateFromMap(h.App.ConfigSnapshot().ReportTemplate)
	if tmpl == nil {
		tmpl = report.DefaultTemplate()
	}
	latest.AccountRaw = publicAccountRaw(latest.AccountRaw)
	if prev != nil {
		prev.AccountRaw = publicAccountRaw(prev.AccountRaw)
	}
	sections := report.BuildSections(gdb, latest, prev, publicTemplate(tmpl))
	for i := range sections {
		for j := range sections[i].Tokens {
			for k := range sections[i].Tokens[j].Metrics {
				m := &sections[i].Tokens[j].Metrics[k]
				if !numericValue(m.Value) {
					m.Value = "—"
				}
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"date": latest.SnapshotDate, "sections": sections})
}
