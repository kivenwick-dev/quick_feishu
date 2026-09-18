package model

import (
	"time"

	"gorm.io/datatypes"
)

type Snapshot struct {
	ID            uint   `gorm:"primaryKey"`
	SnapshotDate  string `gorm:"index"`
	AccountRaw    datatypes.JSON
	TokenListRaw  datatypes.JSON
	TokenUsageRaw datatypes.JSON
	AccountQuota  int64
	AccountUsed   int64
	RequestCount  int64
	CreatedAt     time.Time
}

type TokenSnapshot struct {
	ID           uint `gorm:"primaryKey"`
	SnapshotID   uint `gorm:"index"`
	TokenID      int
	TokenName    string
	UsageRaw     datatypes.JSON
	ListRaw      datatypes.JSON
	UsedQuota    int64
	RemainQuota  int64
	TotalUsed    int64
	TotalGranted int64
}

type SendLog struct {
	ID         uint `gorm:"primaryKey"`
	SendTime   time.Time
	Date       string
	Success    bool
	ErrorMsg   string
	FeishuResp string
}
