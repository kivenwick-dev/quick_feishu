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

// Save 将采集结果写入快照表
func Save(gdb *gorm.DB, date string, res *Result) error {
	accountRaw, _ := json.Marshal(res.Account)
	tokenListRaw, _ := json.Marshal(res.TokenList)
	usageMap := map[string]*api.TokenUsageData{}
	for id, u := range res.Usages {
		usageMap[fmt.Sprintf("%d", id)] = u
	}
	usageRaw, _ := json.Marshal(usageMap)

	snap := &model.Snapshot{
		SnapshotDate:  date,
		AccountRaw:    accountRaw,
		TokenListRaw:  tokenListRaw,
		TokenUsageRaw: usageRaw,
	}
	if res.Account != nil {
		snap.AccountQuota = res.Account.Quota
		snap.AccountUsed = res.Account.UsedQuota
		snap.RequestCount = res.Account.RequestCount
	}
	if err := gdb.Create(snap).Error; err != nil {
		return err
	}
	for _, it := range res.TokenList.Items {
		ts := &model.TokenSnapshot{
			SnapshotID:  snap.ID,
			TokenID:     it.ID,
			TokenName:   it.Name,
			UsedQuota:   it.UsedQuota,
			RemainQuota: it.RemainQuota,
		}
		listRaw, _ := json.Marshal(it)
		ts.ListRaw = listRaw
		if u, ok := res.Usages[it.ID]; ok {
			ts.TotalUsed = u.TotalUsed
			ts.TotalGranted = u.TotalGranted
			ts.UsageRaw = datatypes.JSON(u.Raw)
		}
		if err := gdb.Create(ts).Error; err != nil {
			return err
		}
	}
	return nil
}
