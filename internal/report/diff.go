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

// DiffResult 单字段差值结果
type DiffResult struct {
	Field  string
	Label  string
	Diff   bool
	Value  string // 展示值（差值或当前值）
	IsDiff bool   // 是否实际计算了差值
}

// computeDiff 计算字段差值。早/晚快照可为 nil。
// account 源从 AccountRaw 提取（方案A：AccountRaw 为全字段 data 对象，字段路径为顶层）。
func computeDiff(early, late *model.Snapshot, f Field, label string) (DiffResult, error) {
	var lateVal, earlyVal interface{}
	var ok bool
	res := DiffResult{Field: f.Field, Label: label, Diff: f.Diff}
	if late != nil {
		lateVal, ok = extractField(late.AccountRaw, f.Field)
		if !ok {
			return res, fmt.Errorf("field %s not found", f.Field)
		}
	}
	if !f.Diff {
		res.Value = formatVal(lateVal)
		return res, nil
	}
	if early == nil {
		res.Value = formatVal(lateVal)
		res.IsDiff = false
		return res, nil
	}
	earlyVal, _ = extractField(early.AccountRaw, f.Field)
	// 数值相减，非数值显示晚值
	ln, lok := toFloat(lateVal)
	en, eok := toFloat(earlyVal)
	if lok && eok {
		res.Value = formatNum(ln - en)
		res.IsDiff = true
	} else {
		res.Value = formatVal(lateVal)
		res.IsDiff = false
	}
	return res, nil
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
