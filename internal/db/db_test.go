package db

import (
	"os"
	"path/filepath"
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

func TestInitCreatesDataDir(t *testing.T) {
	base := t.TempDir()
	nested := filepath.Join(base, "does", "not", "exist")
	if _, err := os.Stat(nested); !os.IsNotExist(err) {
		t.Fatalf("precondition: dir should not exist")
	}
	if _, err := Init(nested); err != nil {
		t.Fatalf("Init should create the data dir: %v", err)
	}
	if info, err := os.Stat(nested); err != nil || !info.IsDir() {
		t.Fatalf("data dir not created: %v", err)
	}
}
