package db

import (
	"gorm.io/gorm"
	"quick-feishu/internal/model"
)

// LatestSnapshot 取最近一条快照
func LatestSnapshot(gdb *gorm.DB) (*model.Snapshot, error) {
	var s model.Snapshot
	err := gdb.Order("snapshot_date DESC, id DESC").First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// SnapshotByDate 按日期取快照
func SnapshotByDate(gdb *gorm.DB, date string) (*model.Snapshot, error) {
	var s model.Snapshot
	err := gdb.Where("snapshot_date = ?", date).First(&s).Error
	return &s, err
}

// SnapshotBefore 取指定日期前（含）最近一条快照
func SnapshotBefore(gdb *gorm.DB, date string) (*model.Snapshot, error) {
	var s model.Snapshot
	err := gdb.Where("snapshot_date <= ?", date).Order("snapshot_date DESC, id DESC").First(&s).Error
	return &s, err
}

// TokenSnapshots 取某快照下的令牌列表
func TokenSnapshots(gdb *gorm.DB, snapshotID uint) ([]model.TokenSnapshot, error) {
	var list []model.TokenSnapshot
	err := gdb.Where("snapshot_id = ?", snapshotID).Find(&list).Error
	return list, err
}

// SnapshotsInRange 取日期区间内的快照（升序）
func SnapshotsInRange(gdb *gorm.DB, from, to string) ([]model.Snapshot, error) {
	var list []model.Snapshot
	err := gdb.Where("snapshot_date >= ? AND snapshot_date <= ?", from, to).
		Order("snapshot_date ASC").Find(&list).Error
	return list, err
}

// ListSnapshots 分页列出快照
func ListSnapshots(gdb *gorm.DB, page, size int) ([]model.Snapshot, int64, error) {
	var list []model.Snapshot
	var total int64
	gdb.Model(&model.Snapshot{}).Count(&total)
	err := gdb.Order("snapshot_date DESC, id DESC").Offset((page - 1) * size).Limit(size).Find(&list).Error
	return list, total, err
}

// ListSendLogs 分页列出发送日志
func ListSendLogs(gdb *gorm.DB, page, size int) ([]model.SendLog, int64, error) {
	var list []model.SendLog
	var total int64
	gdb.Model(&model.SendLog{}).Count(&total)
	err := gdb.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&list).Error
	return list, total, err
}
