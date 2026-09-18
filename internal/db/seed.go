package db

import (
	"gorm.io/gorm"
	"quick-feishu/internal/model"
)

type dictSeed struct {
	path  string
	typ   string
	label string
	desc  string
}

var accountSeeds = []dictSeed{
	{"id", "int", "账号ID", ""},
	{"username", "string", "用户名", ""},
	{"display_name", "string", "显示名", ""},
	{"role", "int", "角色", "1=用户 2=管理员"},
	{"status", "int", "状态", "1=启用"},
	{"quota", "int", "总配额", ""},
	{"used_quota", "int", "已用配额", ""},
	{"request_count", "int", "请求次数", ""},
	{"group_id", "int", "分组ID", ""},
	{"group", "string", "分组名", ""},
	{"created_at", "int", "创建时间", "Unix时间戳"},
	{"email", "string", "邮箱", ""},
}

var tokenSeeds = []dictSeed{
	{"id", "int", "令牌ID", ""},
	{"user_id", "int", "所属用户ID", ""},
	{"key", "string", "令牌Key", "用于usage接口认证"},
	{"status", "int", "状态", "1=启用"},
	{"name", "string", "令牌名称", ""},
	{"created_time", "int", "创建时间", ""},
	{"accessed_time", "int", "最后访问时间", ""},
	{"expired_time", "int", "过期时间", "-1=永不过期"},
	{"remain_quota", "int", "剩余配额", ""},
	{"unlimited_quota", "bool", "无限配额", ""},
	{"model_limits_enabled", "bool", "模型限制开关", ""},
	{"model_limits", "string", "模型限制", ""},
	{"allow_ips", "string", "允许IP", ""},
	{"used_quota", "int", "已用配额", ""},
	{"group_ids", "array", "分组ID列表", ""},
	{"group", "string", "分组名", ""},
}

var usageSeeds = []dictSeed{
	{"expires_at", "int", "过期时间", "0=未设置"},
	{"model_limits", "object", "模型限制", ""},
	{"model_limits_enabled", "bool", "模型限制开关", ""},
	{"name", "string", "令牌名称", ""},
	{"object", "string", "对象类型", "固定 token_usage"},
	{"total_available", "int", "可用总量", ""},
	{"total_granted", "int", "授予总量", ""},
	{"total_used", "int", "累计已用", ""},
	{"unlimited_quota", "bool", "无限配额", ""},
}

// SeedDicts 仅当字典表为空时填充内置字段
func SeedDicts(gdb *gorm.DB) error {
	if err := seedIfEmpty(gdb, &model.DictAccountField{}, accountSeeds,
		func(p, t, l, d string) interface{} {
			return &model.DictAccountField{FieldPath: p, FieldType: t, Label: l, Description: d, IsDefault: true}
		}); err != nil {
		return err
	}
	if err := seedIfEmpty(gdb, &model.DictTokenField{}, tokenSeeds,
		func(p, t, l, d string) interface{} {
			return &model.DictTokenField{FieldPath: p, FieldType: t, Label: l, Description: d, IsDefault: true}
		}); err != nil {
		return err
	}
	if err := seedIfEmpty(gdb, &model.DictUsageField{}, usageSeeds,
		func(p, t, l, d string) interface{} {
			return &model.DictUsageField{FieldPath: p, FieldType: t, Label: l, Description: d, IsDefault: true}
		}); err != nil {
		return err
	}
	return nil
}

func seedIfEmpty(gdb *gorm.DB, dest interface{}, seeds []dictSeed, make func(p, t, l, d string) interface{}) error {
	var count int64
	if err := gdb.Model(dest).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	for _, s := range seeds {
		if err := gdb.Create(make(s.path, s.typ, s.label, s.desc)).Error; err != nil {
			return err
		}
	}
	return nil
}
