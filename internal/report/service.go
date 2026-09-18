package report

import (
	"fmt"

	"gorm.io/gorm"
	"quick-feishu/internal/db"
	"quick-feishu/internal/feishu"
	"quick-feishu/internal/model"
)

// ExecuteReport 取最新两日快照，按模板计算差值生成卡片，推送到飞书并记录日志。
// 返回发送日志（无论成败），供调用方展示。
func ExecuteReport(gdb *gorm.DB, latest, prev *model.Snapshot, tmpl *Template, webhookURL string, retryTimes int) (*model.SendLog, error) {
	if latest == nil {
		return nil, fmt.Errorf("no snapshots yet")
	}
	sections := []SectionResult{}
	for _, sec := range tmpl.Sections {
		var secRes []DiffResult
		switch sec.Source {
		case "account":
			for _, f := range sec.Fields {
				res, err := ComputeDiffAccount(prev, latest, f, f.Field)
				if err != nil {
					continue
				}
				secRes = append(secRes, res)
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
			for _, tok := range tokens {
				for _, f := range sec.Fields {
					secRes = append(secRes, tokenDiff(prevTokens, tok, f))
				}
			}
		}
		if len(secRes) > 0 {
			sections = append(sections, SectionResult{Name: sec.Name, Fields: secRes})
		}
	}
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

func tokenDiff(prevTokens []model.TokenSnapshot, tok model.TokenSnapshot, f Field) DiffResult {
	res := DiffResult{Field: f.Field, Diff: f.Diff, Value: formatVal(nil)}
	var prev *model.TokenSnapshot
	for i := range prevTokens {
		if prevTokens[i].TokenID == tok.TokenID {
			prev = &prevTokens[i]
			break
		}
	}
	switch f.Field {
	case "total_used":
		res.Label = "累计已用"
		res.Value = numDiff(prevTotal(prev), tok.TotalUsed)
	case "remain_quota":
		res.Label = "剩余配额"
		res.Value = formatNum(float64(tok.RemainQuota))
	case "used_quota":
		res.Label = "已用配额"
		res.Value = numDiff(prevUsed(prev), tok.UsedQuota)
	default:
		res.Value = formatVal(nil)
	}
	return res
}

func prevTotal(prev *model.TokenSnapshot) int64 {
	if prev != nil {
		return prev.TotalUsed
	}
	return 0
}

func prevUsed(prev *model.TokenSnapshot) int64 {
	if prev != nil {
		return prev.UsedQuota
	}
	return 0
}

func numDiff(prevVal, cur int64) string {
	return formatNum(float64(cur - prevVal))
}
