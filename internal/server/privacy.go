package server

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"

	"quick-feishu/internal/collector"
	"quick-feishu/internal/model"
	"quick-feishu/internal/report"
)

// 只公开具有数量意义的业务指标，数字形式的身份、状态和时间不属于统计指标。
func publicMetricFields(source string) []string {
	switch source {
	case "account":
		return []string{"quota", "used_quota", "balance_usd", "used_usd", "request_count", "aff_count", "aff_quota", "aff_history_quota", "top_up_rebate_count", "withdrawn_quota", "invoice_returned_quota", "support_ticket_cap"}
	case "usage", "token":
		return []string{"total_available", "total_available_usd", "total_granted", "total_granted_usd", "total_used", "total_used_usd", "remaining_percent", "used_quota", "remain_quota"}
	}
	return nil
}

func publicMetric(source, field string) bool {
	for _, candidate := range publicMetricFields(source) {
		if candidate == field {
			return true
		}
	}
	return false
}

func numericValue(value string) bool {
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, "%")
	n, err := strconv.ParseFloat(value, 64)
	return err == nil && !math.IsNaN(n) && !math.IsInf(n, 0)
}

func currencyValue(value string) bool {
	normalized := strings.TrimSpace(value)
	normalized = strings.TrimPrefix(normalized, "+")
	normalized = strings.TrimPrefix(normalized, "-")
	normalized = strings.TrimPrefix(normalized, "$")
	normalized = strings.ReplaceAll(normalized, ",", "")
	return numericValue(normalized)
}

func publicMetricValue(field, value string) bool {
	switch field {
	case "balance_usd", "used_usd", "total_available_usd", "total_granted_usd", "total_used_usd":
		return currencyValue(value)
	default:
		return numericValue(value)
	}
}

func metricDisplayValue(value string) bool {
	return publicMetricValue("balance_usd", value) || numericValue(value) || value == "true" || value == "false"
}

func enablePublicCurrency(f *report.Field) {
	switch f.Field {
	case "balance_usd", "used_usd", "total_available", "total_granted", "total_used":
		on := true
		f.Currency = &on
	}
}

func setPublicCurrency(f *report.Field, enabled bool) {
	if !enabled {
		switch f.Field {
		case "balance_usd", "used_usd", "total_available", "total_granted", "total_used":
			off := false
			f.Currency = &off
		}
		return
	}
	enablePublicCurrency(f)
}

func publicHistory(h *report.History) *report.History {
	fields := []report.HistoryField{}
	for _, f := range h.Fields {
		if publicMetric(h.Source, f.Path) {
			fields = append(fields, f)
		}
	}
	h.Fields = fields
	for i := range h.Rows {
		row := &h.Rows[i]
		values, deltas := map[string]string{}, map[string]string{}
		for _, f := range fields {
			if v, ok := row.Values[f.Path]; ok && publicMetricValue(f.Path, v) {
				values[f.Path] = v
			}
			if d, ok := row.Deltas[f.Path]; ok && publicMetricValue(f.Path, d) {
				deltas[f.Path] = d
			}
		}
		row.Values, row.Deltas = values, deltas
	}
	return h
}

func publicDashboardMetricFields(source string) []string {
	fields := []string{}
	for _, field := range publicMetricFields(source) {
		if source != "account" && strings.HasSuffix(field, "_usd") {
			continue
		}
		fields = append(fields, field)
	}
	return fields
}

func publicTemplate(t *report.Template, currency bool) *report.Template {
	out := *t
	out.Sections = []report.Section{}
	for _, s := range t.Sections {
		section := s
		section.Fields = []report.Field{}
		for _, f := range s.Fields {
			if publicMetric(s.Source, f.Field) {
				f.Diff = true // 看板的统计指标始终计算增减，不依赖日报模板开关。
				setPublicCurrency(&f, currency)
				section.Fields = append(section.Fields, f)
			}
		}
		// 分区和已有排序沿用模板；看板补齐该来源的其他统计指标。
		seen := map[string]bool{}
		for _, f := range section.Fields {
			seen[f.Field] = true
		}
		for _, field := range publicDashboardMetricFields(s.Source) {
			if !seen[field] {
				f := report.Field{Field: field, Diff: true}
				setPublicCurrency(&f, currency)
				section.Fields = append(section.Fields, f)
			}
		}
		out.Sections = append(out.Sections, section)
	}
	return &out
}

// 在进入模板渲染前移除原始 JSON，避免模板或异常类型带出身份和凭据。
func publicAccountRaw(raw []byte) []byte {
	var values map[string]interface{}
	_ = json.Unmarshal(raw, &values)
	safe := map[string]interface{}{}
	for key, value := range values {
		if publicMetric("account", key) {
			if _, ok := value.(float64); ok {
				safe[key] = value
			}
		}
	}
	encoded, _ := json.Marshal(safe)
	return encoded
}

type snapshotView struct {
	CapturedAt time.Time `json:"captured_at"`
	ID         uint      `json:"id"`
	Date       string    `json:"snapshot_date"`
	Quota      int64     `json:"account_quota"`
	Used       int64     `json:"account_used"`
	Requests   int64     `json:"request_count"`
}

func publicSnapshot(s *model.Snapshot) *snapshotView {
	if s == nil {
		return nil
	}
	return &snapshotView{s.CreatedAt, s.ID, s.SnapshotDate, s.AccountQuota, s.AccountUsed, s.RequestCount}
}
func publicSnapshots(list []model.Snapshot) []*snapshotView {
	out := []*snapshotView{}
	for _, s := range list {
		out = append(out, publicSnapshot(&s))
	}
	return out
}

type logView struct {
	ID       uint      `json:"id"`
	SendTime time.Time `json:"send_time"`
	Date     string    `json:"date"`
	Success  bool      `json:"success"`
	Error    string    `json:"error_msg"`
}

func publicLogs(list []model.SendLog) []logView {
	out := []logView{}
	for _, l := range list {
		message := ""
		if !l.Success {
			message = "发送失败，请检查服务端配置或日志"
		}
		out = append(out, logView{l.ID, l.SendTime, l.Date, l.Success, message})
	}
	return out
}

type issueView struct {
	Scope     string `json:"scope"`
	TokenName string `json:"token_name"`
	Kind      string `json:"kind"`
	Status    int    `json:"status"`
	Detail    string `json:"detail"`
}

// publicIssues 输出脱敏后的采集问题；不含令牌 key。
func publicIssues(issues []collector.Issue) []issueView {
	out := []issueView{}
	for _, is := range issues {
		out = append(out, issueView{is.Scope, is.TokenName, is.Kind, is.Status, is.Detail})
	}
	return out
}
