package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGeneratesDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.App.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.App.Port)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("default config should be written: %v", err)
	}
}

func TestLoadReadsExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("account:\n  user_id: \"123\"\n"), 0644)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Account.UserID != "123" {
		t.Errorf("expected 123, got %s", cfg.Account.UserID)
	}
}

func TestEnvOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.Setenv("QR_USER_ID", "env-user")
	defer os.Unsetenv("QR_USER_ID")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Account.UserID != "env-user" {
		t.Errorf("expected env override, got %s", cfg.Account.UserID)
	}
	if !cfg.EnvOverridden()["user_id"] {
		t.Error("user_id should be marked env-overridden")
	}
}

func TestSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := Default()
	cfg.Account.UserID = "999"
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}
	reloaded, _ := Load(path)
	if reloaded.Account.UserID != "999" {
		t.Errorf("expected 999 after save, got %s", reloaded.Account.UserID)
	}
}
