package app

import (
	"path/filepath"
	"testing"

	"quick-feishu/internal/config"
	"quick-feishu/internal/db"
)

func TestStartRestartAndNextRuns(t *testing.T) {
	gdb, err := db.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	a := New(cfg, gdb, filepath.Join(t.TempDir(), "config.yaml"))

	if err := a.StartScheduler(); err != nil {
		t.Fatal(err)
	}
	if got := len(a.NextRuns()); got != 2 {
		t.Fatalf("next runs = %d, want 2", got)
	}

	cfg.Schedule.SnapshotTime = "01:23"
	if err := a.Restart(); err != nil {
		t.Fatal(err)
	}
	if got := len(a.NextRuns()); got != 2 {
		t.Fatalf("after restart next runs = %d, want 2", got)
	}
}

func TestClientRebuiltOnRestart(t *testing.T) {
	gdb, err := db.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "config.yaml")

	cfg := config.Default()
	cfg.Account.UserID = "1"
	cfg.Account.SystemToken = "a"
	a := New(cfg, gdb, path)
	if a.Client().UserID != "1" {
		t.Fatalf("initial client user = %s", a.Client().UserID)
	}

	next := config.Default()
	next.Account.UserID = "2"
	next.Account.SystemToken = "b"
	if err := a.SaveConfig(next); err != nil {
		t.Fatal(err)
	}
	if err := a.Restart(); err != nil {
		t.Fatal(err)
	}
	if got := a.Client().UserID; got != "2" {
		t.Errorf("client not rebuilt after restart: %s", got)
	}
	if got := a.Client().SystemToken; got != "b" {
		t.Errorf("client token not rebuilt: %s", got)
	}
}

func TestSaveSettingsPreservesWriteOnlyCredentials(t *testing.T) {
	for _, key := range []string{"QR_USER_ID", "QR_SYSTEM_TOKEN", "QR_FEISHU_WEBHOOK"} {
		t.Setenv(key, "")
	}
	gdb, err := db.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Account.UserID = "existing-account"
	cfg.Account.SystemToken = "existing-token"
	cfg.Account.APIBase = "https://example.test"
	cfg.Feishu.WebhookURL = "https://example.test/webhook"
	a := New(cfg, gdb, filepath.Join(t.TempDir(), "config.yaml"))
	expectedAccount, expectedFeishu := cfg.Account, cfg.Feishu
	incoming := *cfg
	incoming.Account = config.AccountConfig{}
	incoming.Feishu.WebhookURL = ""
	incoming.Schedule.ReportTime = "12:34"
	if err := a.SaveConfig(&incoming); err != nil {
		t.Fatal(err)
	}
	if cfg.Account != expectedAccount || cfg.Feishu != expectedFeishu {
		t.Fatal("blank settings overwrote credentials")
	}
	if cfg.Schedule.ReportTime != "12:34" {
		t.Fatal("non-secret setting not saved")
	}
	incoming = *cfg
	incoming.Account.SystemToken = "replacement-token"
	if err := a.SaveConfig(&incoming); err != nil {
		t.Fatal(err)
	}
	if cfg.Account.SystemToken != "replacement-token" {
		t.Fatal("new credential not saved")
	}
}
