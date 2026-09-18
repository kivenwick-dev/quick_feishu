package db

import (
	"testing"
)

func TestInitMigrations(t *testing.T) {
	dir := t.TempDir()
	gdb, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := gdb.Table("snapshots").Count(&count).Error; err != nil {
		t.Fatalf("snapshots table not queryable: %v", err)
	}
	for _, tbl := range []string{"token_snapshots", "send_logs", "dict_account_fields", "dict_token_fields", "dict_usage_fields"} {
		if err := gdb.Table(tbl).Count(&count).Error; err != nil {
			t.Fatalf("%s table missing: %v", tbl, err)
		}
	}
}
