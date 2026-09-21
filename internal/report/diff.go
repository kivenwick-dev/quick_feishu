package report

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"quick-feishu/internal/model"
)

const quotaPerUSD = 500000.0

func isUSDField(path string) bool {
	return path == "balance_usd" || path == "used_usd"
}

func isTokenUSDField(path string) bool {
	return path == "total_available" || path == "total_used" || path == "total_granted"
}

func tokenUSDSourceField(path string) (string, bool) {
	switch path {
	case "total_available_usd":
		return "total_available", true
	case "total_used_usd":
		return "total_used", true
	case "total_granted_usd":
		return "total_granted", true
	default:
		return "", false
	}
}

func isCurrencyField(path string) bool {
	if isUSDField(path) {
		return true
	}
	_, ok := tokenUSDSourceField(path)
	return ok
}

func isConsumptionDeltaField(path string) bool {
	switch path {
	case "quota", "balance_usd", "total_available", "total_available_usd", "remain_quota":
		return true
	default:
		return false
	}
}

func deltaLabel(path string) string {
	if isConsumptionDeltaField(path) {
		return "消耗"
	}
	return "变动"
}

func deltaAmount(path string, late, early float64) float64 {
	if isConsumptionDeltaField(path) {
		return early - late
	}
	return late - early
}

func snapshotAccountField(s *model.Snapshot, path string) (interface{}, bool) {
	if s == nil {
		return nil, false
	}
	switch path {
	case "balance_usd":
		return s.AccountQuota, true
	case "used_usd":
		return s.AccountUsed, true
	default:
		return extractField(s.AccountRaw, path)
	}
}

// extractField 从原始 JSON 中提取字段值（支持嵌套 . 路径）
func extractField(raw []byte, path string) (interface{}, bool) {
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, false
	}
	cur := interface{}(m)
	for _, seg := range splitPath(path) {
		mm, ok := cur.(map[string]interface{})
		if !ok {
			return nil, false
		}
		cur, ok = mm[seg]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

func splitPath(p string) []string {
	var segs []string
	cur := ""
	for _, r := range p {
		if r == '.' {
			segs = append(segs, cur)
			cur = ""
		} else {
			cur += string(r)
		}
	}
	segs = append(segs, cur)
	return segs
}

// DiffResult 单字段结果：Value 为今日值；若启用差值且可计算，Delta 为带符号变动值。
type DiffResult struct {
	Field      string `json:"field"`
	Label      string `json:"label"`
	Value      string `json:"value"`       // 今日值（始终展示）
	Delta      string `json:"delta"`       // 变动值（带符号），仅当 IsDiff 为 true
	DeltaLabel string `json:"delta_label"` // 变动含义，如“变动”或“消耗”
	IsDiff     bool   `json:"is_diff"`
}

// ComputeDiffAccount 对账号快照字段取值并计算差值。
// account 源从 AccountRaw 提取（方案A：AccountRaw 为全字段 data 对象，字段路径为顶层）。
func ComputeDiffAccount(early, late *model.Snapshot, f Field, label string) (DiffResult, error) {
	res := DiffResult{Field: f.Field, Label: label}
	if late == nil {
		return res, fmt.Errorf("no late snapshot")
	}
	lateVal, ok := snapshotAccountField(late, f.Field)
	if !ok {
		return res, fmt.Errorf("field %s not found", f.Field)
	}
	var earlyVal interface{}
	if early != nil {
		earlyVal, _ = snapshotAccountField(early, f.Field)
	}
	return DiffFieldValue(f.Field, label, lateVal, earlyVal, f.Diff, f.CurrencyEnabled(), int64(quotaPerUSD)), nil
}

// DiffValue 通用取值+差值：Value 始终为晚值；wantDiff 且两值均可转数值时计算带符号 Delta。
func DiffValue(label string, lateVal, earlyVal interface{}, wantDiff bool) DiffResult {
	return DiffDisplayValue(label, lateVal, lateVal, earlyVal, wantDiff)
}

func DiffDisplayValue(label string, displayVal, lateDeltaVal, earlyVal interface{}, wantDiff bool) DiffResult {
	res := DiffResult{Label: label, Value: formatVal(displayVal)}
	if !wantDiff {
		return res
	}
	ln, lok := toFloat(lateDeltaVal)
	en, eok := toFloat(earlyVal)
	if lok && eok {
		res.Delta = signedNum(ln - en)
		res.IsDiff = true
	}
	return res
}

func DiffFieldValue(field, label string, lateVal, earlyVal interface{}, wantDiff bool, currency bool, quotaPerUnit int64) DiffResult {
	return DiffFieldDisplayValue(field, label, lateVal, lateVal, earlyVal, wantDiff, currency, quotaPerUnit)
}

func DiffFieldDisplayValue(field, label string, displayVal, lateDeltaVal, earlyVal interface{}, wantDiff bool, currency bool, quotaPerUnit int64) DiffResult {
	if !isUSDField(field) || !currency || quotaPerUnit <= 0 {
		res := DiffResult{Field: field, Label: label, Value: formatVal(displayVal), DeltaLabel: deltaLabel(field)}
		if !wantDiff {
			return res
		}
		ln, lok := toFloat(lateDeltaVal)
		en, eok := toFloat(earlyVal)
		if lok && eok {
			res.Delta = signedNum(deltaAmount(field, ln, en))
			res.IsDiff = true
		}
		return res
	}
	rate := float64(quotaPerUnit)
	res := DiffResult{Field: field, Label: label, Value: formatUSDValue(displayVal, rate), DeltaLabel: deltaLabel(field)}
	if !wantDiff {
		return res
	}
	ln, lok := toFloat(lateDeltaVal)
	en, eok := toFloat(earlyVal)
	if lok && eok {
		res.Delta = signedCurrencyDelta(deltaAmount(field, ln, en) / rate)
		res.IsDiff = true
	}
	return res
}

func DiffTokenUSDDisplayValue(field, label string, displayVal, lateQuotaVal, earlyQuotaVal interface{}, quotaPerUnit int64, wantDiff bool, currency bool) DiffResult {
	if !isTokenUSDField(field) || !currency || quotaPerUnit <= 0 {
		res := DiffResult{Field: field, Label: label, Value: formatVal(displayVal), DeltaLabel: deltaLabel(field)}
		if !wantDiff {
			return res
		}
		ln, lok := toFloat(lateQuotaVal)
		en, eok := toFloat(earlyQuotaVal)
		if lok && eok {
			res.Delta = signedNum(deltaAmount(field, ln, en))
			res.IsDiff = true
		}
		return res
	}
	rate := float64(quotaPerUnit)
	res := DiffResult{Field: field, Label: label, Value: formatUSDValue(displayVal, rate), DeltaLabel: deltaLabel(field)}
	if !wantDiff {
		return res
	}
	ln, lok := toFloat(lateQuotaVal)
	en, eok := toFloat(earlyQuotaVal)
	if lok && eok {
		res.Delta = signedCurrencyDelta(deltaAmount(field, ln, en) / rate)
		res.IsDiff = true
	}
	return res
}

// signedNum 带符号格式化：正数加 +，负数自带 -，0 显示 0
func signedNum(f float64) string {
	if f > 0 {
		return "+" + formatNum(f)
	}
	return formatNum(f)
}

func signedFieldNum(field string, f float64) string {
	if !isCurrencyField(field) {
		return signedNum(f)
	}
	return signedCurrencyDelta(f / quotaPerUSD)
}

func signedCurrencyDelta(f float64) string {
	if f > 0 {
		return "+" + formatUSD(f)
	}
	if f < 0 {
		return "-" + formatUSD(-f)
	}
	return "$0.00"
}

func toFloat(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(t, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func formatNum(f float64) string {
	return strconv.FormatFloat(f, 'f', 0, 64)
}

func formatVal(v interface{}) string {
	if v == nil {
		return "-"
	}
	switch t := v.(type) {
	case float64:
		return strconv.FormatFloat(t, 'f', 0, 64)
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func formatFieldVal(field string, v interface{}) string {
	if !isCurrencyField(field) {
		return formatVal(v)
	}
	f, ok := toFloat(v)
	if !ok {
		return "-"
	}
	return formatUSD(f / quotaPerUSD)
}

func formatUSDValue(v interface{}, quotaPerUnit float64) string {
	f, ok := toFloat(v)
	if !ok || quotaPerUnit <= 0 {
		return "-"
	}
	return formatUSD(f / quotaPerUnit)
}

func formatUSD(f float64) string {
	value := strconv.FormatFloat(f, 'f', 2, 64)
	parts := strings.Split(value, ".")
	whole := parts[0]
	negative := strings.HasPrefix(whole, "-")
	if negative {
		whole = strings.TrimPrefix(whole, "-")
	}
	var chunks []string
	for len(whole) > 3 {
		chunks = append([]string{whole[len(whole)-3:]}, chunks...)
		whole = whole[:len(whole)-3]
	}
	chunks = append([]string{whole}, chunks...)
	prefix := "$"
	if negative {
		prefix = "-$"
	}
	return prefix + strings.Join(chunks, ",") + "." + parts[1]
}
