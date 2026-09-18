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
	Raw          json.RawMessage `json:"-"` // 原始 data 对象（全字段）
}

// GetAccount 获取账号信息，返回 data 对象全字段原始 JSON
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
	var envelope struct {
		Data    json.RawMessage `json:"data"`
		Message string          `json:"message"`
		Success bool            `json:"success"`
	}
	if err := parseJSON(body, &envelope); err != nil {
		return nil, body, err
	}
	var d AccountData
	if err := parseJSON(envelope.Data, &d); err != nil {
		return nil, envelope.Data, err
	}
	d.Raw = envelope.Data
	return &d, envelope.Data, nil
}
