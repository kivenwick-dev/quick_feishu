package db

import (
	"testing"

	"quick-feishu/internal/model"
)

func TestSeedDicts(t *testing.T) {
	dir := t.TempDir()
	gdb, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := SeedDicts(gdb); err != nil {
		t.Fatal(err)
	}
	var ac []model.DictAccountField
	gdb.Find(&ac)
	if len(ac) != len(accountSeeds) {
		t.Errorf("account seeds = %d, want %d", len(ac), len(accountSeeds))
	}
	// 二次调用不重复
	if err := SeedDicts(gdb); err != nil {
		t.Fatal(err)
	}
	gdb.Find(&ac)
	if len(ac) != len(accountSeeds) {
		t.Errorf("second seed should be no-op, got %d", len(ac))
	}
	// 校验 token 和 usage 表
	var tc []model.DictTokenField
	gdb.Find(&tc)
	if len(tc) != len(tokenSeeds) {
		t.Errorf("token seeds = %d, want %d", len(tc), len(tokenSeeds))
	}
	var uc []model.DictUsageField
	gdb.Find(&uc)
	if len(uc) != len(usageSeeds) {
		t.Errorf("usage seeds = %d, want %d", len(uc), len(usageSeeds))
	}
	// 校验 IsDefault 标记
	if !ac[0].IsDefault {
		t.Error("first account field should be IsDefault=true")
	}
}

func TestSeedDictsUpgradesExistingDictionary(t *testing.T) {
	gdb, err := Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []model.DictAccountField{
		{FieldPath: "username", Label: "自定义用户名"},
		{FieldPath: "aff_code", Label: "aff_code"},
		{FieldPath: "email", Label: ""},
	} {
		if err := gdb.Create(&field).Error; err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := SeedDicts(gdb); err != nil {
			t.Fatal(err)
		}
	}
	labels := DictLabels(gdb, "account")
	for path, want := range map[string]string{
		"username": "自定义用户名", "aff_code": "邀请码", "email": "邮箱",
		"bulk_topup_discount_disabled": "禁用批量充值折扣",
	} {
		if labels[path] != want {
			t.Errorf("%s = %q, want %q", path, labels[path], want)
		}
	}
	var count int64
	gdb.Model(&model.DictAccountField{}).Count(&count)
	if count != int64(len(accountSeeds)) {
		t.Errorf("count = %d, want %d", count, len(accountSeeds))
	}
}
