package feishu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// sleepFn 便于测试注入，生产默认 time.Sleep
var sleepFn = time.Sleep

type Client struct {
	WebhookURL string
	RetryTimes int
	HTTPClient *http.Client
}

func NewClient(webhookURL string, retryTimes int) *Client {
	return &Client{
		WebhookURL: webhookURL,
		RetryTimes: retryTimes,
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// SendCard 发送卡片/消息 JSON，失败重试（2s/4s/8s 退避）
func (c *Client) SendCard(cardJSON []byte) (string, error) {
	var lastErr error
	var respBody string
	for i := 0; i <= c.RetryTimes; i++ {
		respBody, lastErr = c.post(cardJSON)
		if lastErr == nil {
			return respBody, nil
		}
		if i < c.RetryTimes {
			sleepFn(time.Duration(1<<uint(i)) * 2 * time.Second)
		}
	}
	return respBody, lastErr
}

func (c *Client) post(cardJSON []byte) (string, error) {
	req, err := http.NewRequest(http.MethodPost, c.WebhookURL, bytes.NewReader(cardJSON))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return string(body), fmt.Errorf("feishu http %d: %s", resp.StatusCode, string(body))
	}
	var r struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(body, &r); err == nil && r.Code != 0 {
		return string(body), fmt.Errorf("feishu code %d: %s", r.Code, r.Msg)
	}
	return string(body), nil
}
