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
	{"balance_usd", "float", "当前余额", "当前额度按 500,000 quota = $1.00 USD 换算"},
	{"used_usd", "float", "历史消耗", "累计已用额度按 500,000 quota = $1.00 USD 换算"},
	{"request_count", "int", "请求次数", ""},
	{"group_id", "int", "分组ID", ""},
	{"group", "string", "分组名", ""},
	{"created_at", "int", "创建时间", "Unix时间戳"},
	{"email", "string", "邮箱", ""},
	{"DeletedAt", "string", "删除时间", ""},
	{"access_token", "string", "访问令牌", ""},
	{"account", "string", "账户", ""},
	{"aff_code", "string", "邀请码", ""},
	{"aff_count", "int", "邀请人数", ""},
	{"aff_history_quota", "int", "历史邀请奖励配额", ""},
	{"aff_quota", "int", "可用邀请奖励配额", ""},
	{"alipay_account", "string", "支付宝账号", ""},
	{"alipay_real_name", "string", "支付宝实名", ""},
	{"bulk_topup_discount_disabled", "bool", "禁用批量充值折扣", ""},
	{"cost_price_marker", "string", "成本价格标记", ""},
	{"github_id", "string", "GitHub 用户标识", ""},
	{"google_id", "string", "谷歌用户标识", ""},
	{"hongming_id", "string", "鸿鸣用户标识", ""},
	{"inviter_id", "int", "邀请人标识", ""},
	{"invoice_returned_quota", "int", "已退回发票配额", ""},
	{"linux_do_id", "string", "Linux DO 用户标识", ""},
	{"oidc_id", "string", "统一身份认证标识", ""},
	{"original_password", "string", "原始密码", ""},
	{"owner_sub_station_id", "int", "所属子站标识", ""},
	{"parent_id", "int", "上级账号标识", ""},
	{"password", "string", "密码", ""},
	{"password_version", "int", "密码版本", ""},
	{"phone", "string", "手机号", ""},
	{"real_name", "string", "真实姓名", ""},
	{"setting", "string", "个人设置", ""},
	{"star_level", "int", "星级", ""},
	{"stripe_customer", "string", "Stripe 客户标识", ""},
	{"support_online", "bool", "在线客服开关", ""},
	{"support_online_at", "int", "在线客服时间", ""},
	{"support_ticket_cap", "int", "工单数量上限", ""},
	{"telegram_id", "string", "Telegram 用户标识", ""},
	{"top_up_rebate_count", "int", "充值返利次数", ""},
	{"usdt_address", "string", "泰达币收款地址", ""},
	{"usdt_chain", "string", "泰达币网络", ""},
	{"verification_code", "string", "验证码", ""},
	{"wechat_id", "string", "微信标识", ""},
	{"withdrawn_quota", "int", "已提现配额", ""},
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
	{"remaining_percent", "int", "剩余用量", "可用总量 ÷ 授予总量，按整数百分比展示"},
	{"unlimited_quota", "bool", "无限配额", ""},
}

// SeedDicts 增量补齐内置字段，保留用户自定义名称。
func SeedDicts(gdb *gorm.DB) error {
	if err := seedMissing(gdb, &model.DictAccountField{}, accountSeeds,
		func(p, t, l, d string) interface{} {
			return &model.DictAccountField{FieldPath: p, FieldType: t, Label: l, Description: d, IsDefault: true}
		}); err != nil {
		return err
	}
	if err := seedMissing(gdb, &model.DictTokenField{}, tokenSeeds,
		func(p, t, l, d string) interface{} {
			return &model.DictTokenField{FieldPath: p, FieldType: t, Label: l, Description: d, IsDefault: true}
		}); err != nil {
		return err
	}
	if err := seedMissing(gdb, &model.DictUsageField{}, usageSeeds,
		func(p, t, l, d string) interface{} {
			return &model.DictUsageField{FieldPath: p, FieldType: t, Label: l, Description: d, IsDefault: true}
		}); err != nil {
		return err
	}
	return nil
}

func seedMissing(gdb *gorm.DB, dest interface{}, seeds []dictSeed, make func(p, t, l, d string) interface{}) error {
	for _, s := range seeds {
		var count int64
		if err := gdb.Model(dest).Where("field_path = ?", s.path).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := gdb.Create(make(s.path, s.typ, s.label, s.desc)).Error; err != nil {
				return err
			}
			continue
		}
		// 修复空名称和以原始字段名占位的旧记录，不覆盖已有中文名称。
		if err := gdb.Model(dest).Where("field_path = ? AND (label = '' OR label IS NULL OR label = ?)", s.path, s.path).
			Update("label", s.label).Error; err != nil {
			return err
		}
	}
	return nil
}
