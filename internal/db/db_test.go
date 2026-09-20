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

func TestPath(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	wantAccount := filepath.Join(dir, "accounts", "42", "quick-feishu.db")
	if got := Path(dir, "42"); got != wantAccount {
		t.Errorf("Path(42) = %s, want %s", got, wantAccount)
	}
	wantLegacy := filepath.Join(dir, "quick-feishu.db")
	if got := Path(dir, ""); got != wantLegacy {
		t.Errorf("Path(empty) = %s, want %s", got, wantLegacy)
	}
}

func TestMigrateLegacyMovesOldDb(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "quick-feishu.db")
	if err := os.WriteFile(legacy, []byte("legacy"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := MigrateLegacy(dir, "42"); err != nil {
		t.Fatal(err)
	}
	target := Path(dir, "42")
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("target missing: %v", err)
	}
	if string(got) != "legacy" {
		t.Fatalf("target content = %q", got)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("legacy should be gone, stat err = %v", err)
	}
}

func TestMigrateLegacyDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "quick-feishu.db")
	target := Path(dir, "42")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte("legacy"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("target"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := MigrateLegacy(dir, "42"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "target" {
		t.Fatalf("target overwritten: %q", got)
	}
	if _, err := os.Stat(legacy); err != nil {
		t.Fatalf("legacy should remain when target exists: %v", err)
	}
}

func TestMigrateLegacyEmptyUserIDIsNoop(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "quick-feishu.db")
	if err := os.WriteFile(legacy, []byte("legacy"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := MigrateLegacy(dir, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(legacy); err != nil {
		t.Fatalf("legacy moved unexpectedly: %v", err)
	}
}

func TestMigrateLegacyMissingLegacyIsNoop(t *testing.T) {
	dir := t.TempDir()
	if err := MigrateLegacy(dir, "42"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(Path(dir, "42")); !os.IsNotExist(err) {
		t.Fatalf("no target should be created when legacy is missing, stat err = %v", err)
	}
}
