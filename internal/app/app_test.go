package app

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"quick-feishu/internal/config"
	"quick-feishu/internal/db"
	"quick-feishu/internal/model"
)

func clearCredentialEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"QR_USER_ID", "QR_SYSTEM_TOKEN", "QR_FEISHU_WEBHOOK"} {
		t.Setenv(key, "")
	}
}

// stubAPIServer 对所有请求返回 200 {}，用于验证客户端与调度器重启。
func stubAPIServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestStartRestartAndNextRuns(t *testing.T) {
	clearCredentialEnv(t)
	cfg := config.Default()
	a, err := New(cfg, t.TempDir(), filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
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
	clearCredentialEnv(t)
	srv := stubAPIServer(t)
	path := filepath.Join(t.TempDir(), "config.yaml")

	cfg := config.Default()
	cfg.Account.UserID = "1"
	cfg.Account.SystemToken = "a"
	cfg.Account.APIBase = srv.URL
	a, err := New(cfg, t.TempDir(), path)
	if err != nil {
		t.Fatal(err)
	}
	if a.Client().UserID != "1" {
		t.Fatalf("initial client user = %s", a.Client().UserID)
	}

	next := config.Default()
	next.Account.UserID = "2"
	next.Account.SystemToken = "b"
	next.Account.APIBase = srv.URL
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
	clearCredentialEnv(t)
	cfg := config.Default()
	cfg.Account.UserID = "existing-account"
	cfg.Account.SystemToken = "existing-token"
	cfg.Account.APIBase = "https://example.test"
	cfg.Feishu.WebhookURL = "https://example.test/webhook"
	a, err := New(cfg, t.TempDir(), filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
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

func TestRestartSwitchesDatabase(t *testing.T) {
	clearCredentialEnv(t)
	srv := stubAPIServer(t)
	dir := t.TempDir()
	path := filepath.Join(t.TempDir(), "config.yaml")

	cfg := config.Default()
	cfg.Account.UserID = "account-a"
	cfg.Account.APIBase = srv.URL
	a, err := New(cfg, dir, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.DB().Create(&model.Snapshot{SnapshotDate: "2026-09-18", AccountUsed: 100}).Error; err != nil {
		t.Fatal(err)
	}

	next := config.Default()
	next.Account.UserID = "account-b"
	next.Account.APIBase = srv.URL
	if err := a.SaveConfig(next); err != nil {
		t.Fatal(err)
	}
	if err := a.Restart(); err != nil {
		t.Fatal(err)
	}

	if want := db.Path(dir, "account-b"); a.currentDBPath != want {
		t.Fatalf("currentDBPath = %s, want %s", a.currentDBPath, want)
	}
	var count int64
	a.DB().Model(&model.Snapshot{}).Where("account_used = ?", 100).Count(&count)
	if count != 0 {
		t.Fatalf("account A data leaked into account B db: count = %d", count)
	}
	a.DB().Model(&model.Snapshot{}).Count(&count)
	if count != 0 {
		t.Fatalf("restart must leave immediate collection to its caller, snapshots = %d", count)
	}
}

func TestRestartSameUserKeepsDatabase(t *testing.T) {
	clearCredentialEnv(t)
	dir := t.TempDir()
	path := filepath.Join(t.TempDir(), "config.yaml")

	cfg := config.Default()
	cfg.Account.UserID = "same-user"
	a, err := New(cfg, dir, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.DB().Create(&model.Snapshot{SnapshotDate: "2026-09-18", AccountUsed: 100}).Error; err != nil {
		t.Fatal(err)
	}
	if err := a.Config.Save(path); err != nil {
		t.Fatal(err)
	}
	if err := a.Restart(); err != nil {
		t.Fatal(err)
	}
	var count int64
	a.DB().Model(&model.Snapshot{}).Where("account_used = ?", 100).Count(&count)
	if count != 1 {
		t.Fatalf("data lost on same-user restart: count = %d", count)
	}
}

func TestEmptyUserIDUsesLegacyPath(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	a, err := New(cfg, dir, filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "quick-feishu.db"); a.currentDBPath != want {
		t.Fatalf("currentDBPath = %s, want %s", a.currentDBPath, want)
	}
}

func TestConfigSnapshotIsCopy(t *testing.T) {
	cfg := config.Default()
	cfg.Account.UserID = "u1"
	a, err := New(cfg, t.TempDir(), filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	snap := a.ConfigSnapshot()
	snap.Account.UserID = "mutated"
	if got := a.ConfigSnapshot().Account.UserID; got != "u1" {
		t.Fatalf("ConfigSnapshot must return a copy, got %s", got)
	}
}
