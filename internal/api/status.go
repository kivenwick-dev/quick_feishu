package api

import "encoding/json"

// StatusData contains the public billing display settings used by QuickRouter.
type StatusData struct {
	QuotaPerUnit      int64 `json:"quota_per_unit"`
	DisplayInCurrency bool  `json:"display_in_currency"`
}

// GetStatus reads the platform-wide quota conversion settings. The endpoint is public.
func (c *Client) GetStatus() (*StatusData, error) {
	body, err := c.doGet("/api/status", nil, nil)
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := parseJSON(body, &envelope); err != nil {
		return nil, err
	}
	var status StatusData
	if err := parseJSON(envelope.Data, &status); err != nil {
		return nil, err
	}
	return &status, nil
}
