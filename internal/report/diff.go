package report

import (
	"encoding/json"
	"fmt"
	"strconv"

	"quick-feishu/internal/model"
)

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
	Field  string `json:"field"`
	Label  string `json:"label"`
	Value  string `json:"value"` // 今日值（始终展示）
	Delta  string `json:"delta"` // 变动值（带符号），仅当 IsDiff 为 true
	IsDiff bool   `json:"is_diff"`
}

// ComputeDiffAccount 对账号快照字段取值并计算差值。
// account 源从 AccountRaw 提取（方案A：AccountRaw 为全字段 data 对象，字段路径为顶层）。
func ComputeDiffAccount(early, late *model.Snapshot, f Field, label string) (DiffResult, error) {
	res := DiffResult{Field: f.Field, Label: label}
	if late == nil {
		return res, fmt.Errorf("no late snapshot")
	}
	lateVal, ok := extractField(late.AccountRaw, f.Field)
	if !ok {
		return res, fmt.Errorf("field %s not found", f.Field)
	}
	var earlyVal interface{}
	if early != nil {
		earlyVal, _ = extractField(early.AccountRaw, f.Field)
	}
	return DiffValue(label, lateVal, earlyVal, f.Diff), nil
}

// DiffValue 通用取值+差值：Value 始终为晚值；wantDiff 且两值均可转数值时计算带符号 Delta。
func DiffValue(label string, lateVal, earlyVal interface{}, wantDiff bool) DiffResult {
	res := DiffResult{Label: label, Value: formatVal(lateVal)}
	if !wantDiff {
		return res
	}
	ln, lok := toFloat(lateVal)
	en, eok := toFloat(earlyVal)
	if lok && eok {
		res.Delta = signedNum(ln - en)
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
