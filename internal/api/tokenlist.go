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
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
	Total    int         `json:"total"`
	Items    []TokenItem `json:"items"`
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
	Raw                json.RawMessage `json:"-"`
}

// GetTokenList 获取令牌列表（分页拉全）
func (c *Client) GetTokenList() (*TokenListData, []byte, error) {
	headers := map[string]string{
		"new-api-user":  c.UserID,
		"Authorization": "Bearer " + c.SystemToken,
	}
	page := 0
	all := &TokenListData{Items: []TokenItem{}}
	var lastRaw []byte
	for {
		query := map[string]string{"p": fmt.Sprintf("%d", page), "size": "100"}
		body, err := WithRetry(3, func() ([]byte, error) {
			return c.doGet("/api/token/", query, headers)
		})
		if err != nil {
			return all, lastRaw, err
		}
		var resp TokenListResp
		if err := parseJSON(body, &resp); err != nil {
			return all, body, err
		}
		all.Items = append(all.Items, resp.Data.Items...)
		all.Total = resp.Data.Total
		all.PageSize = resp.Data.PageSize
		lastRaw = body
		if len(resp.Data.Items) == 0 || (page+1)*resp.Data.PageSize >= resp.Data.Total {
			break
		}
		page++
	}
	return all, lastRaw, nil
}
