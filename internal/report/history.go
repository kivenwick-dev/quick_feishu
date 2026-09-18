package report

import (
	"encoding/json"
	"sort"
	"time"

	"gorm.io/gorm"
	"quick-feishu/internal/db"
	"quick-feishu/internal/model"
)

// HistoryField 历史表列定义
type HistoryField struct {
	Path  string `json:"path"`
	Label string `json:"label"`
}

// HistoryRow 每条采集记录，Deltas 相对上一次可用记录的带符号变动。
type HistoryRow struct {
	CapturedAt time.Time         `json:"captured_at"`
	ID         uint              `json:"id"`
	Date       string            `json:"date"`
	Values     map[string]string `json:"values"`
	Deltas     map[string]string `json:"deltas"`
}

// History 某接口的每日指标矩阵
type History struct {
	Source string         `json:"source"`
	Fields []HistoryField `json:"fields"`
	Rows   []HistoryRow   `json:"rows"`
}

func topKeys(raw []byte) []string {
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// buildFields 合并字典字段与快照中实际出现的字段（字典优先，额外字段追加在后）
func buildFields(dict []HistoryField, raws [][]byte) []HistoryField {
	seen := map[string]bool{}
	out := []HistoryField{}
	for _, f := range dict {
		if !seen[f.Path] {
			seen[f.Path] = true
			out = append(out, f)
		}
	}
	var extras []string
	for _, raw := range raws {
		for _, k := range topKeys(raw) {
			if !seen[k] {
				seen[k] = true
				extras = append(extras, k)
			}
		}
	}
	sort.Strings(extras)
	for _, k := range extras {
		out = append(out, HistoryField{Path: k, Label: k})
	}
	return out
}

func dictHistoryFields(gdb *gorm.DB, source string) []HistoryField {
	info := db.DictFieldList(gdb, source)
	out := make([]HistoryField, 0, len(info))
	for _, f := range info {
		out = append(out, HistoryField{Path: f.Path, Label: f.Label})
	}
	return out
}

func fillRow(fields []HistoryField, get func(string) (interface{}, bool), prev map[string]interface{}) (HistoryRow, map[string]interface{}) {
	row := HistoryRow{Values: map[string]string{}, Deltas: map[string]string{}}
	cur := map[string]interface{}{}
	for _, f := range fields {
		v, ok := get(f.Path)
		if !ok {
			continue
		}
		cur[f.Path] = v
		row.Values[f.Path] = formatVal(v)
		if prev != nil {
			if pv, ok := prev[f.Path]; ok {
				ln, lok := toFloat(v)
				pn, pok := toFloat(pv)
				if lok && pok {
					row.Deltas[f.Path] = signedNum(ln - pn)
				}
			}
		}
	}
	return row, cur
}

func recentSnapshots(gdb *gorm.DB, limit int) []model.Snapshot {
	snaps, _ := db.SnapshotsInRange(gdb, "0000-01-01", "9999-12-31")
	if limit > 0 && len(snaps) > limit {
		snaps = snaps[len(snaps)-limit:]
	}
	return snaps
}

// BuildAccountHistory 构建账号信息的每日指标矩阵
func BuildAccountHistory(gdb *gorm.DB, limit int) *History {
	snaps := recentSnapshots(gdb, limit)
	raws := make([][]byte, 0, len(snaps))
	for _, s := range snaps {
		raws = append(raws, s.AccountRaw)
	}
	fields := buildFields(dictHistoryFields(gdb, "account"), raws)

	h := &History{Source: "account", Fields: fields, Rows: []HistoryRow{}}
	var prev map[string]interface{}
	for _, s := range snaps {
		snap := s
		row, cur := fillRow(fields, func(path string) (interface{}, bool) {
			return extractField(snap.AccountRaw, path)
		}, prev)
		row.ID = s.ID
		row.Date = s.SnapshotDate
		row.CapturedAt = s.CreatedAt
		prev = cur
		h.Rows = append(h.Rows, row)
	}
	return h
}

// BuildUsageHistory 构建指定令牌的使用情况每日指标矩阵
func BuildUsageHistory(gdb *gorm.DB, tokenID int, limit int) *History {
	snaps := recentSnapshots(gdb, limit)
	type entry struct {
		date       string
		capturedAt time.Time
		sid        uint
		tok        model.TokenSnapshot
	}
	var entries []entry
	var raws [][]byte
	for _, s := range snaps {
		toks, _ := db.TokenSnapshots(gdb, s.ID)
		for _, t := range toks {
			if t.TokenID == tokenID {
				entries = append(entries, entry{date: s.SnapshotDate, capturedAt: s.CreatedAt, sid: s.ID, tok: t})
				raws = append(raws, []byte(t.UsageRaw), []byte(t.ListRaw))
				break
			}
		}
	}
	dict := append(dictHistoryFields(gdb, "usage"), dictHistoryFields(gdb, "token")...)
	fields := buildFields(dict, raws)

	h := &History{Source: "usage", Fields: fields, Rows: []HistoryRow{}}
	var prev map[string]interface{}
	for _, e := range entries {
		ent := e
		row, cur := fillRow(fields, func(path string) (interface{}, bool) {
			return extractTokenField(ent.tok, path)
		}, prev)
		row.ID = e.sid
		row.Date = e.date
		row.CapturedAt = e.capturedAt
		prev = cur
		h.Rows = append(h.Rows, row)
	}
	return h
}
