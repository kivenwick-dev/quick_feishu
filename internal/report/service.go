package report

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"gorm.io/gorm"
	"quick-feishu/internal/db"
	"quick-feishu/internal/feishu"
	"quick-feishu/internal/model"
)

const remainingPercentField = "remaining_percent"

// TemplateFromMap 将配置中的 report_template map 转换为 Template；为空时返回默认模板。
func TemplateFromMap(m map[string]interface{}) (*Template, error) {
	if len(m) == 0 {
		return DefaultTemplate(), nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return DefaultTemplate(), err
	}
	t, err := ParseTemplate(b)
	if err != nil {
		return DefaultTemplate(), err
	}
	return t, nil
}

// labelOf 返回字段的中文名；字典没有则回退字段路径
func labelOf(labels map[string]string, field string) string {
	if l := labels[field]; l != "" {
		return l
	}
	return field
}

// extractTokenField 从令牌快照中取值：先 usage 原始 JSON，再列表原始 JSON
func extractTokenField(tok model.TokenSnapshot, path string) (interface{}, bool) {
	if v, ok := extractField(tok.UsageRaw, path); ok {
		return v, true
	}
	if v, ok := extractField(tok.ListRaw, path); ok {
		return v, true
	}
	return nil, false
}

// BuildSections 按模板将快照数据组装为分区结果（不涉及网络）。
func BuildSections(gdb *gorm.DB, latest, prev *model.Snapshot, tmpl *Template) []SectionResult {
	return BuildSectionsWithAccountOverrides(gdb, latest, prev, tmpl, nil)
}

// BuildSectionsWithAccountOverrides 按模板组装分区结果，可对账号展示值做运行时覆盖。
// 覆盖值只影响字段展示；差值仍使用 latest 与 prev 快照计算。
func BuildSectionsWithAccountOverrides(gdb *gorm.DB, latest, prev *model.Snapshot, tmpl *Template, accountOverrides map[string]interface{}) []SectionResult {
	return BuildSectionsWithOverrides(gdb, latest, prev, tmpl, accountOverrides, nil, 0)
}

// BuildSectionsWithOverrides 按模板组装分区结果，可覆盖账号与令牌展示值。
// 覆盖值只影响字段展示；差值仍使用 latest 与 prev 快照计算。
func BuildSectionsWithOverrides(gdb *gorm.DB, latest, prev *model.Snapshot, tmpl *Template, accountOverrides map[string]interface{}, tokenOverrides map[int]map[string]interface{}, quotaPerUnit int64) []SectionResult {
	if quotaPerUnit <= 0 {
		quotaPerUnit = int64(quotaPerUSD)
	}
	accountLabels := db.DictLabels(gdb, "account")
	usageLabels := db.DictLabels(gdb, "usage")
	tokenLabels := db.DictLabels(gdb, "token")

	sections := []SectionResult{}
	for _, sec := range tmpl.Sections {
		switch sec.Source {
		case "account":
			s := SectionResult{Name: sec.Name}
			for _, f := range sec.Fields {
				if f.Field == "" {
					continue
				}
				lateVal, ok := snapshotAccountField(latest, f.Field)
				if !ok {
					continue
				}
				displayVal := lateVal
				if isUSDField(f.Field) && accountOverrides != nil {
					if override, ok := accountOverrides[f.Field]; ok {
						displayVal = override
					}
				}
				var earlyVal interface{}
				if prev != nil {
					earlyVal, _ = snapshotAccountField(prev, f.Field)
				}
				s.Fields = append(s.Fields, DiffFieldDisplayValue(f.Field, labelOf(accountLabels, f.Field), displayVal, lateVal, earlyVal, f.Diff, f.CurrencyEnabled(), quotaPerUnit))
			}
			if len(s.Fields) > 0 {
				sections = append(sections, s)
			}

		case "usage":
			tokens, err := db.TokenSnapshots(gdb, latest.ID)
			if err != nil {
				continue
			}
			if len(tokens) == 0 && len(tokenOverrides) > 0 {
				tokens = tokenSnapshotsFromOverrides(tokenOverrides)
			}
			prevTokens, _ := db.TokenSnapshots(gdb, prevID(prev))
			prevByID := map[int]model.TokenSnapshot{}
			for _, pt := range prevTokens {
				prevByID[pt.TokenID] = pt
			}
			s := SectionResult{Name: sec.Name}
			for _, tok := range tokens {
				var pt *model.TokenSnapshot
				if p, ok := prevByID[tok.TokenID]; ok {
					pt = &p
				}
				node := TokenNode{Name: tok.TokenName}
				if tokenUnlimited(tok, tokenOverrides) {
					node.Metrics = append(node.Metrics, Metric{Label: "无限配额", Value: "true"})
					s.Tokens = append(s.Tokens, node)
					continue
				}
				hasRemainingPercent := false
				for _, f := range sec.Fields {
					if f.Field == "" || f.Field == "name" || f.Field == "unlimited_quota" {
						continue // name 用作二级节点标题，不再作为指标
					}
					if f.Field == remainingPercentField {
						if metric, ok := remainingPercentMetric(tok, tokenOverrides); ok {
							node.Metrics = append(node.Metrics, metric)
							hasRemainingPercent = true
						}
						continue
					}
					lateVal, ok := extractTokenField(tok, f.Field)
					if !ok {
						lateVal, ok = tokenDisplayValue(tok, f.Field, tokenOverrides)
					}
					if !ok {
						continue
					}
					displayVal := lateVal
					if isTokenUSDField(f.Field) && tokenOverrides != nil {
						if byField, ok := tokenOverrides[tok.TokenID]; ok {
							if override, ok := byField[f.Field]; ok {
								displayVal = override
							}
						}
					}
					var earlyVal interface{}
					if pt != nil {
						earlyVal, _ = extractTokenField(*pt, f.Field)
					}
					label := labelOf(usageLabels, f.Field)
					if usageLabels[f.Field] == "" {
						label = labelOf(tokenLabels, f.Field)
					}
					res := DiffTokenUSDDisplayValue(f.Field, label, displayVal, lateVal, earlyVal, quotaPerUnit, f.Diff, f.CurrencyEnabled())
					node.Metrics = append(node.Metrics, Metric{
						Label:      res.Label,
						Value:      res.Value,
						Delta:      res.Delta,
						DeltaLabel: res.DeltaLabel,
						HasDelta:   res.IsDiff,
					})
				}
				if !hasRemainingPercent {
					if metric, ok := remainingPercentMetric(tok, tokenOverrides); ok {
						node.Metrics = append(node.Metrics, metric)
					}
				}
				s.Tokens = append(s.Tokens, node)
			}
			if len(s.Tokens) > 0 {
				sections = append(sections, s)
			}
		}
	}
	return sections
}

func tokenSnapshotsFromOverrides(tokenOverrides map[int]map[string]interface{}) []model.TokenSnapshot {
	ids := make([]int, 0, len(tokenOverrides))
	for id := range tokenOverrides {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	tokens := make([]model.TokenSnapshot, 0, len(ids))
	for _, id := range ids {
		name := fmt.Sprintf("token-%d", id)
		if raw, ok := tokenOverrides[id]["name"]; ok {
			if s, ok := raw.(string); ok && s != "" {
				name = s
			}
		}
		tokens = append(tokens, model.TokenSnapshot{TokenID: id, TokenName: name})
	}
	return tokens
}

func remainingPercentMetric(tok model.TokenSnapshot, tokenOverrides map[int]map[string]interface{}) (Metric, bool) {
	if tokenUnlimited(tok, tokenOverrides) {
		return Metric{}, false
	}
	available, ok := tokenDisplayValue(tok, "total_available", tokenOverrides)
	if !ok {
		return Metric{}, false
	}
	granted, ok := tokenDisplayValue(tok, "total_granted", tokenOverrides)
	if !ok {
		return Metric{}, false
	}
	availableNum, ok := toFloat(available)
	if !ok {
		return Metric{}, false
	}
	grantedNum, ok := toFloat(granted)
	if !ok || grantedNum <= 0 {
		return Metric{}, false
	}
	percent := int(math.Round(availableNum / grantedNum * 100))
	return Metric{Label: "剩余用量", Value: fmt.Sprintf("%d%%", percent)}, true
}

func tokenUnlimited(tok model.TokenSnapshot, tokenOverrides map[int]map[string]interface{}) bool {
	v, ok := tokenDisplayValue(tok, "unlimited_quota", tokenOverrides)
	if ok {
		switch t := v.(type) {
		case bool:
			if t {
				return true
			}
		case string:
			if t == "true" || t == "1" {
				return true
			}
		default:
			f, ok := toFloat(t)
			if ok && f != 0 {
				return true
			}
		}
	}
	available, ok := tokenDisplayValue(tok, "total_available", tokenOverrides)
	if !ok {
		return false
	}
	availableNum, ok := toFloat(available)
	return ok && availableNum < 0
}

func tokenDisplayValue(tok model.TokenSnapshot, field string, tokenOverrides map[int]map[string]interface{}) (interface{}, bool) {
	if tokenOverrides != nil {
		if byField, ok := tokenOverrides[tok.TokenID]; ok {
			if override, ok := byField[field]; ok {
				return override, true
			}
		}
	}
	return extractTokenField(tok, field)
}

// ExecuteReport 取最新两日快照，按模板计算差值生成卡片，推送到飞书并记录日志。
// 返回发送日志（无论成败），供调用方展示。
func ExecuteReport(gdb *gorm.DB, latest, prev *model.Snapshot, tmpl *Template, webhookURL string, retryTimes int) (*model.SendLog, error) {
	return ExecuteReportWithAccountOverrides(gdb, latest, prev, tmpl, webhookURL, retryTimes, nil)
}

func ExecuteReportWithAccountOverrides(gdb *gorm.DB, latest, prev *model.Snapshot, tmpl *Template, webhookURL string, retryTimes int, accountOverrides map[string]interface{}) (*model.SendLog, error) {
	return ExecuteReportWithOverrides(gdb, latest, prev, tmpl, webhookURL, retryTimes, accountOverrides, nil, 0)
}

func ExecuteReportWithOverrides(gdb *gorm.DB, latest, prev *model.Snapshot, tmpl *Template, webhookURL string, retryTimes int, accountOverrides map[string]interface{}, tokenOverrides map[int]map[string]interface{}, quotaPerUnit int64) (*model.SendLog, error) {
	if latest == nil {
		return nil, fmt.Errorf("no snapshots yet")
	}
	sections := BuildSectionsWithOverrides(gdb, latest, prev, tmpl, accountOverrides, tokenOverrides, quotaPerUnit)
	card, err := BuildCard(tmpl, latest.SnapshotDate, sections)
	if err != nil {
		return nil, err
	}
	cardJSON, _ := card.ToJSON()
	client := feishu.NewClient(webhookURL, retryTimes)
	resp, sendErr := client.SendCard(cardJSON)
	log := &model.SendLog{Date: latest.SnapshotDate, Success: sendErr == nil, FeishuResp: resp}
	if sendErr != nil {
		log.ErrorMsg = sendErr.Error()
	}
	if err := gdb.Create(log).Error; err != nil {
		return log, err
	}
	return log, sendErr
}

func prevID(prev *model.Snapshot) uint {
	if prev != nil {
		return prev.ID
	}
	return 0
}
