package collector

import (
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"quick-feishu/internal/api"
	"quick-feishu/internal/model"
)

type Result struct {
	Account   *api.AccountData
	TokenList *api.TokenListData
	Usages    map[int]*api.TokenUsageData // tokenID -> usage
	Errors    []string
}

// Collect 依次调用三接口，单接口失败不阻断其余
func Collect(c *api.Client) *Result {
	res := &Result{Usages: map[int]*api.TokenUsageData{}, Errors: []string{}}

	account, _, err := c.GetAccount()
	if err != nil {
		res.Errors = append(res.Errors, "account: "+err.Error())
	} else {
		res.Account = account
	}

	list, _, err := c.GetTokenList()
	if err != nil {
		res.Errors = append(res.Errors, "tokenlist: "+err.Error())
		res.TokenList = &api.TokenListData{Items: []api.TokenItem{}}
	} else {
		res.TokenList = list
		for _, it := range list.Items {
			usage, _, uerr := c.GetTokenUsage(it.Key)
			if uerr != nil {
				res.Errors = append(res.Errors, "usage("+it.Name+"): "+uerr.Error())
				continue
			}
			res.Usages[it.ID] = usage
		}
	}
	return res
}

// Save 将采集结果（全字段原始 JSON）写入快照表。
// 按日期覆盖：同一天重复采集会替换旧快照，避免产生多条同日快照。
func Save(gdb *gorm.DB, date string, res *Result) error {
	usageMap := map[string]json.RawMessage{}
	for id, u := range res.Usages {
		if u != nil && len(u.Raw) > 0 {
			usageMap[fmt.Sprintf("%d", id)] = u.Raw
		}
	}
	usageRaw, _ := json.Marshal(usageMap)

	return gdb.Transaction(func(tx *gorm.DB) error {
		// 1) 删除同日旧快照及其令牌快照
		var olds []model.Snapshot
		if err := tx.Where("snapshot_date = ?", date).Find(&olds).Error; err != nil {
			return err
		}
		for _, o := range olds {
			if err := tx.Where("snapshot_id = ?", o.ID).Delete(&model.TokenSnapshot{}).Error; err != nil {
				return err
			}
			if err := tx.Delete(&model.Snapshot{}, o.ID).Error; err != nil {
				return err
			}
		}

		// 2) 写入新快照
		snap := &model.Snapshot{SnapshotDate: date}
		if res.Account != nil {
			snap.AccountRaw = datatypes.JSON(res.Account.Raw)
			snap.AccountQuota = res.Account.Quota
			snap.AccountUsed = res.Account.UsedQuota
			snap.RequestCount = res.Account.RequestCount
		}
		if res.TokenList != nil {
			snap.TokenListRaw = datatypes.JSON(res.TokenList.Raw)
		}
		snap.TokenUsageRaw = datatypes.JSON(usageRaw)
		if err := tx.Create(snap).Error; err != nil {
			return err
		}
		if res.TokenList != nil {
			for _, it := range res.TokenList.Items {
				ts := &model.TokenSnapshot{
					SnapshotID:  snap.ID,
					TokenID:     it.ID,
					TokenName:   it.Name,
					UsedQuota:   it.UsedQuota,
					RemainQuota: it.RemainQuota,
				}
				if len(it.Raw) > 0 {
					ts.ListRaw = datatypes.JSON(it.Raw)
				}
				if u, ok := res.Usages[it.ID]; ok && u != nil {
					ts.TotalUsed = u.TotalUsed
					ts.TotalGranted = u.TotalGranted
					if len(u.Raw) > 0 {
						ts.UsageRaw = datatypes.JSON(u.Raw)
					}
				}
				if err := tx.Create(ts).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
