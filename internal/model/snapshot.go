package model

import (
	"time"

	"gorm.io/datatypes"
)

type Snapshot struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	SnapshotDate  string         `gorm:"index" json:"snapshot_date"`
	AccountRaw    datatypes.JSON `json:"account_raw"`
	TokenListRaw  datatypes.JSON `json:"token_list_raw"`
	TokenUsageRaw datatypes.JSON `json:"token_usage_raw"`
	AccountQuota  int64          `json:"account_quota"`
	AccountUsed   int64          `json:"account_used"`
	RequestCount  int64          `json:"request_count"`
	CreatedAt     time.Time      `json:"created_at"`
}

type TokenSnapshot struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	SnapshotID   uint           `gorm:"index" json:"snapshot_id"`
	TokenID      int            `json:"token_id"`
	TokenName    string         `json:"token_name"`
	UsageRaw     datatypes.JSON `json:"usage_raw"`
	ListRaw      datatypes.JSON `json:"list_raw"`
	UsedQuota    int64          `json:"used_quota"`
	RemainQuota  int64          `json:"remain_quota"`
	TotalUsed    int64          `json:"total_used"`
	TotalGranted int64          `json:"total_granted"`
}

type SendLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	SendTime   time.Time `json:"send_time"`
	Date       string    `json:"date"`
	Success    bool      `json:"success"`
	ErrorMsg   string    `json:"error_msg"`
	FeishuResp string    `json:"feishu_resp"`
}
