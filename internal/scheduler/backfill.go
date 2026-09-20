package scheduler

import (
	"time"

	"gorm.io/gorm"
	"quick-feishu/internal/api"
	"quick-feishu/internal/collector"
	"quick-feishu/internal/db"
)

// BackfillMissing 自上次快照日期起，补齐到今天（含）之前缺失的每日快照。
// 无任何快照时，仅采集今天。collectFn 用于采集（便于测试注入）。
func BackfillMissing(gdb *gorm.DB, c *api.Client, today string, collectFn func() *collector.Result) error {
	latest, err := db.LatestSnapshot(gdb)
	if err != nil {
		// 无任何快照：补采今天
		return collector.SaveForAccount(gdb, today, collectFn(), c.UserID)
	}
	start, perr := time.Parse("2006-01-02", latest.SnapshotDate)
	if perr != nil {
		return perr
	}
	for d := start.AddDate(0, 0, 1); ; d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		if dateStr > today {
			break
		}
		if err := collector.SaveForAccount(gdb, dateStr, collectFn(), c.UserID); err != nil {
			return err
		}
	}
	return nil
}
