package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const DefaultBase = "https://api.quickrouter.ai"

var sleepFn = time.Sleep

type Client struct {
	BaseURL     string
	HTTPClient  *http.Client
	SystemToken string
	UserID      string
}

func NewClient(baseURL, systemToken, userID string) *Client {
	return &Client{
		BaseURL:     baseURL,
		SystemToken: systemToken,
		UserID:      userID,
		HTTPClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// doGet 发起 GET 请求，extraHeaders 附加到请求头
func (c *Client) doGet(path string, query map[string]string, extraHeaders map[string]string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	for k, v := range query {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func parseJSON(body []byte, v interface{}) error {
	return json.Unmarshal(body, v)
}

// WithRetry 带退避重试执行 fn（retries 次，2s/4s/8s...）
func WithRetry(retries int, fn func() ([]byte, error)) ([]byte, error) {
	var lastErr error
	for i := 0; i <= retries; i++ {
		var body []byte
		body, lastErr = fn()
		if lastErr == nil {
			return body, nil
		}
		if i < retries {
			sleepFn(time.Duration(1<<uint(i)) * 2 * time.Second)
		}
	}
	return nil, lastErr
}
