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
