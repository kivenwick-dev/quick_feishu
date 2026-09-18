package api

import "encoding/json"

type TokenUsageResp struct {
	Data    TokenUsageData `json:"data"`
	Message string         `json:"message"`
	Success bool           `json:"success"`
}

type TokenUsageData struct {
	ExpiresAt          int64           `json:"expires_at"`
	ModelLimits        json.RawMessage `json:"model_limits"`
	ModelLimitsEnabled bool            `json:"model_limits_enabled"`
	Name               string          `json:"name"`
	Object             string          `json:"object"`
	TotalAvailable     int64           `json:"total_available"`
	TotalGranted       int64           `json:"total_granted"`
	TotalUsed          int64           `json:"total_used"`
	UnlimitedQuota     bool            `json:"unlimited_quota"`
	Raw                json.RawMessage `json:"-"`
}

// GetTokenUsage 获取单个令牌使用情况，tokenKey 是令牌自身 key
func (c *Client) GetTokenUsage(tokenKey string) (*TokenUsageData, []byte, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + tokenKey,
	}
	body, err := WithRetry(3, func() ([]byte, error) {
		return c.doGet("/api/usage/token/", nil, headers)
	})
	if err != nil {
		return nil, nil, err
	}
	var resp TokenUsageResp
	if err := parseJSON(body, &resp); err != nil {
		return nil, body, err
	}
	resp.Data.Raw = body
	return &resp.Data, body, nil
}
