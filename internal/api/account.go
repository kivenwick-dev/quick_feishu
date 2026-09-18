package api

import "encoding/json"

type AccountResp struct {
	Data    AccountData `json:"data"`
	Message string      `json:"message"`
	Success bool        `json:"success"`
}

type AccountData struct {
	ID           int             `json:"id"`
	Username     string          `json:"username"`
	DisplayName  string          `json:"display_name"`
	Role         int             `json:"role"`
	Status       int             `json:"status"`
	Quota        int64           `json:"quota"`
	UsedQuota    int64           `json:"used_quota"`
	RequestCount int64           `json:"request_count"`
	GroupID      int             `json:"group_id"`
	Group        string          `json:"group"`
	CreatedAt    int64           `json:"created_at"`
	Raw          json.RawMessage `json:"-"`
}

// GetAccount 获取账号信息，返回原始 JSON 供全量存储
func (c *Client) GetAccount() (*AccountData, []byte, error) {
	headers := map[string]string{
		"new-api-user":  c.UserID,
		"Authorization": "Bearer " + c.SystemToken,
	}
	body, err := WithRetry(3, func() ([]byte, error) {
		return c.doGet("/api/user/self", nil, headers)
	})
	if err != nil {
		return nil, nil, err
	}
	var resp AccountResp
	if err := parseJSON(body, &resp); err != nil {
		return nil, body, err
	}
	resp.Data.Raw = body
	return &resp.Data, body, nil
}
