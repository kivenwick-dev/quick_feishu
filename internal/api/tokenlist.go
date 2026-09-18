package api

import (
	"encoding/json"
	"fmt"
)

type TokenListResp struct {
	Data    TokenListData `json:"data"`
	Message string        `json:"message"`
	Success bool          `json:"success"`
}

type TokenListData struct {
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
	Total    int             `json:"total"`
	Items    []TokenItem     `json:"items"`
	Raw      json.RawMessage `json:"-"` // 汇总的原始 data 对象（全字段）
}

type TokenItem struct {
	ID                 int             `json:"id"`
	UserID             int             `json:"user_id"`
	Key                string          `json:"key"`
	Status             int             `json:"status"`
	Name               string          `json:"name"`
	CreatedTime        int64           `json:"created_time"`
	AccessedTime       int64           `json:"accessed_time"`
	ExpiredTime        int64           `json:"expired_time"`
	RemainQuota        int64           `json:"remain_quota"`
	UnlimitedQuota     bool            `json:"unlimited_quota"`
	ModelLimitsEnabled bool            `json:"model_limits_enabled"`
	ModelLimits        string          `json:"model_limits"`
	AllowIPs           string          `json:"allow_ips"`
	UsedQuota          int64           `json:"used_quota"`
	GroupIDs           []int           `json:"group_ids"`
	Group              string          `json:"group"`
	Raw                json.RawMessage `json:"-"` // 该令牌原始 JSON（全字段）
}

// GetTokenList 获取令牌列表（分页拉全），保留每项全字段原始 JSON
func (c *Client) GetTokenList() (*TokenListData, []byte, error) {
	headers := map[string]string{
		"new-api-user":  c.UserID,
		"Authorization": "Bearer " + c.SystemToken,
	}
	page := 0
	all := &TokenListData{Items: []TokenItem{}}
	var allRaw []json.RawMessage
	var total, pageSize int
	for {
		query := map[string]string{"p": fmt.Sprintf("%d", page), "size": "100"}
		body, err := WithRetry(3, func() ([]byte, error) {
			return c.doGet("/api/token/", query, headers)
		})
		if err != nil {
			return all, nil, err
		}
		var envelope struct {
			Data    json.RawMessage `json:"data"`
			Message string          `json:"message"`
			Success bool            `json:"success"`
		}
		if err := parseJSON(body, &envelope); err != nil {
			return all, body, err
		}
		var dataObj struct {
			Page     int               `json:"page"`
			PageSize int               `json:"page_size"`
			Total    int               `json:"total"`
			Items    []json.RawMessage `json:"items"`
		}
		if err := parseJSON(envelope.Data, &dataObj); err != nil {
			return all, envelope.Data, err
		}
		total = dataObj.Total
		pageSize = dataObj.PageSize
		for _, rawItem := range dataObj.Items {
			var item TokenItem
			if err := parseJSON(rawItem, &item); err != nil {
				return all, envelope.Data, err
			}
			item.Raw = rawItem
			all.Items = append(all.Items, item)
			allRaw = append(allRaw, rawItem)
		}
		if len(dataObj.Items) == 0 || pageSize <= 0 || (page+1)*pageSize >= total {
			break
		}
		page++
	}
	all.Total = total
	all.PageSize = pageSize
	combined, err := json.Marshal(struct {
		Total int               `json:"total"`
		Items []json.RawMessage `json:"items"`
	}{Total: total, Items: allRaw})
	if err != nil {
		return all, nil, err
	}
	all.Raw = combined
	return all, combined, nil
}
