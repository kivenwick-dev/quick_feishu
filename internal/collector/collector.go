package collector

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"quick-feishu/internal/api"
	"quick-feishu/internal/model"
)

// 采集问题分类
const (
	KindQuotaExhausted = "quota_exhausted"
	KindUnauthorized   = "unauthorized"
	KindNetwork        = "network"
	KindServerError    = "server_error"
	KindOther          = "other"
)

// Issue 描述一次采集中的单个失败项。脱敏：不含令牌 key，Detail 仅保留平台 message。
type Issue struct {
	Scope     string `json:"scope"`      // account | tokenlist | usage
	TokenName string `json:"token_name"` // usage 时为令牌名
	Kind      string `json:"kind"`
	Status    int    `json:"status"` // HTTP 状态码；网络错误为 0
	Detail    string `json:"detail"`
}

type Result struct {
	Account   *api.AccountData
	TokenList *api.TokenListData
	Usages    map[int]*api.TokenUsageData
	Issues    []Issue
}

// classify 将一次采集错误归类为 Issue。
func classify(scope, tokenName, secret string, err error) Issue {
	issue := Issue{Scope: scope, TokenName: tokenName, Kind: KindOther}
	var he *api.HTTPError
	if !errors.As(err, &he) {
		issue.Kind = KindNetwork
		issue.Detail = redact(err.Error(), secret, 0)
		return issue
	}
	issue.Status = he.Status
	issue.Detail = redact(platformMessage(he.Body), secret, he.Status)
	if issue.Detail == "" {
		issue.Detail = fmt.Sprintf("HTTP %d", he.Status)
	}
	switch {
	case he.Status == http.StatusUnauthorized && strings.Contains(he.Body, "额度已用尽"):
		issue.Kind = KindQuotaExhausted
	case he.Status == http.StatusUnauthorized || he.Status == http.StatusForbidden:
		issue.Kind = KindUnauthorized
	case he.Status >= 500:
		issue.Kind = KindServerError
	}
	return issue
}

// redact 当明细包含令牌 key 时，替换为不含敏感信息的表述。
func redact(detail, secret string, status int) string {
	if secret != "" && strings.Contains(detail, secret) {
		if status > 0 {
			return fmt.Sprintf("HTTP %d", status)
		}
		return "已隐藏敏感信息"
	}
	return detail
}

// platformMessage 只从 {"error":{"message":"..."}} 取 message，绝不回传原始 body。
func platformMessage(body string) string {
	var envelope struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		return ""
	}
	return envelope.Error.Message
}

// Collect 依次调用三接口，单接口失败不阻断其余
func Collect(c *api.Client) *Result {
	res := &Result{Usages: map[int]*api.TokenUsageData{}, Issues: []Issue{}}

	account, _, err := c.GetAccount()
	if err != nil {
		res.Issues = append(res.Issues, classify("account", "", "", err))
	} else {
		res.Account = account
	}

	list, _, err := c.GetTokenList()
	if err != nil {
		res.Issues = append(res.Issues, classify("tokenlist", "", "", err))
		res.TokenList = &api.TokenListData{Items: []api.TokenItem{}}
	} else {
		res.TokenList = list
		for _, it := range list.Items {
			usage, _, uerr := c.GetTokenUsage(it.Key)
			if uerr != nil {
				res.Issues = append(res.Issues, classify("usage", it.Name, it.Key, uerr))
				continue
			}
			res.Usages[it.ID] = usage
		}
	}
	return res
}

// Save 将采集结果（全字段原始 JSON）写入快照表。
// 每次采集新增记录，以 CreatedAt 记录具体时间，同日采集也保留。
func Save(gdb *gorm.DB, date string, res *Result) error {
	usageMap := map[string]json.RawMessage{}
	for id, u := range res.Usages {
		if u != nil && len(u.Raw) > 0 {
			usageMap[fmt.Sprintf("%d", id)] = u.Raw
		}
	}
	usageRaw, _ := json.Marshal(usageMap)

	return gdb.Transaction(func(tx *gorm.DB) error {
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
