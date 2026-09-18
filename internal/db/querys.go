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

// DictLabels 返回某接口字段字典的 field_path -> label 映射
func DictLabels(gdb *gorm.DB, source string) map[string]string {
	m := map[string]string{}
	switch source {
	case "account":
		var list []model.DictAccountField
		gdb.Find(&list)
		for _, v := range list {
			m[v.FieldPath] = v.Label
		}
	case "token":
		var list []model.DictTokenField
		gdb.Find(&list)
		for _, v := range list {
			m[v.FieldPath] = v.Label
		}
	case "usage":
		var list []model.DictUsageField
		gdb.Find(&list)
		for _, v := range list {
			m[v.FieldPath] = v.Label
		}
	}
	return m
}

// FieldInfo 字段字典项（有序）
type FieldInfo struct {
	Path  string `json:"path"`
	Label string `json:"label"`
}

// DictFieldList 返回某接口字段字典的有序列表（按内置顺序）
func DictFieldList(gdb *gorm.DB, source string) []FieldInfo {
	var out []FieldInfo
	switch source {
	case "account":
		var list []model.DictAccountField
		gdb.Order("id ASC").Find(&list)
		for _, v := range list {
			out = append(out, FieldInfo{Path: v.FieldPath, Label: v.Label})
		}
	case "token":
		var list []model.DictTokenField
		gdb.Order("id ASC").Find(&list)
		for _, v := range list {
			out = append(out, FieldInfo{Path: v.FieldPath, Label: v.Label})
		}
	case "usage":
		var list []model.DictUsageField
		gdb.Order("id ASC").Find(&list)
		for _, v := range list {
			out = append(out, FieldInfo{Path: v.FieldPath, Label: v.Label})
		}
	}
	return out
}

// LatestTokens 返回最近一次快照下的令牌列表
func LatestTokens(gdb *gorm.DB) ([]model.TokenSnapshot, error) {
	latest, err := LatestSnapshot(gdb)
	if err != nil {
		return nil, err
	}
	return TokenSnapshots(gdb, latest.ID)
}
