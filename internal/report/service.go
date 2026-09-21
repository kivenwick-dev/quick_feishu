package report

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
	"quick-feishu/internal/db"
	"quick-feishu/internal/feishu"
	"quick-feishu/internal/model"
)

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
			if !sec.PerToken {
				continue
			}
			tokens, err := db.TokenSnapshots(gdb, latest.ID)
			if err != nil {
				continue
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
				for _, f := range sec.Fields {
					if f.Field == "" || f.Field == "name" {
						continue // name 用作二级节点标题，不再作为指标
					}
					lateVal, ok := extractTokenField(tok, f.Field)
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
				s.Tokens = append(s.Tokens, node)
			}
			if len(s.Tokens) > 0 {
				sections = append(sections, s)
			}
		}
	}
	return sections
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
