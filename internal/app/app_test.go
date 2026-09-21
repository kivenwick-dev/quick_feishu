package app

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/datatypes"
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

func mustReportJSON(t *testing.T, m map[string]interface{}) datatypes.JSON {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return datatypes.JSON(b)
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

func TestRunReportUsesLiveCurrencyValuesButSnapshotDeltas(t *testing.T) {
	clearCredentialEnv(t)
	var webhookBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/self":
			_, _ = w.Write([]byte(`{"data":{"id":827947,"quota":200000000,"used_quota":120000000,"request_count":99},"success":true}`))
		case "/api/status":
			_, _ = w.Write([]byte(`{"data":{"quota_per_unit":500000,"display_in_currency":true},"success":true}`))
		case "/api/token/":
			_, _ = w.Write([]byte(`{"data":{"page":1,"page_size":100,"total":1,"items":[{"id":7,"key":"token-key","name":"live-key"}]},"success":true}`))
		case "/api/usage/token/":
			if r.Header.Get("Authorization") != "Bearer token-key" {
				t.Fatalf("usage Authorization = %q", r.Header.Get("Authorization"))
			}
			_, _ = w.Write([]byte(`{"data":{"name":"live-key","total_available":2000000,"total_used":1500000,"total_granted":3500000},"success":true}`))
		case "/hook":
			body, _ := io.ReadAll(r.Body)
			webhookBody = string(body)
			_, _ = w.Write([]byte(`{"code":0,"msg":"ok"}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	cfg := config.Default()
	cfg.Account.UserID = "827947"
	cfg.Account.APIBase = srv.URL
	cfg.Feishu.WebhookURL = srv.URL + "/hook"
	cfg.ReportTemplate = map[string]interface{}{
		"title":     "测试日报",
		"date_mode": "auto",
		"sections": []interface{}{
			map[string]interface{}{
				"section": "账号概况",
				"source":  "account",
				"fields": []interface{}{
					map[string]interface{}{"field": "used_quota", "diff": true},
					map[string]interface{}{"field": "balance_usd", "diff": true, "currency": true},
					map[string]interface{}{"field": "used_usd", "diff": true, "currency": true},
				},
			},
			map[string]interface{}{
				"section":   "各令牌用量",
				"source":    "usage",
				"per_token": true,
				"fields": []interface{}{
					map[string]interface{}{"field": "total_available", "diff": true, "currency": true},
					map[string]interface{}{"field": "total_used", "diff": true, "currency": true},
					map[string]interface{}{"field": "total_granted", "diff": true, "currency": true},
				},
			},
		},
	}
	a, err := New(cfg, t.TempDir(), filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SeedDicts(a.DB()); err != nil {
		t.Fatal(err)
	}
	prev := &model.Snapshot{
		SnapshotDate: "2026-09-20",
		AccountQuota: 110000000,
		AccountUsed:  50000000,
		AccountRaw:   mustReportJSON(t, map[string]interface{}{"used_quota": float64(50000000)}),
	}
	if err := a.DB().Create(prev).Error; err != nil {
		t.Fatal(err)
	}
	if err := a.DB().Create(&model.TokenSnapshot{
		SnapshotID: prev.ID,
		TokenID:    7,
		TokenName:  "live-key",
		UsageRaw: mustReportJSON(t, map[string]interface{}{
			"total_available": float64(2000000),
			"total_used":      float64(500000),
			"total_granted":   float64(1500000),
		}),
	}).Error; err != nil {
		t.Fatal(err)
	}
	latestSnap := &model.Snapshot{
		SnapshotDate: "2026-09-21",
		AccountQuota: 100000000,
		AccountUsed:  80000000,
		AccountRaw:   mustReportJSON(t, map[string]interface{}{"used_quota": float64(80000000)}),
	}
	if err := a.DB().Create(latestSnap).Error; err != nil {
		t.Fatal(err)
	}
	if err := a.DB().Create(&model.TokenSnapshot{
		SnapshotID: latestSnap.ID,
		TokenID:    7,
		TokenName:  "live-key",
		UsageRaw: mustReportJSON(t, map[string]interface{}{
			"total_available": float64(1500000),
			"total_used":      float64(1000000),
			"total_granted":   float64(2500000),
		}),
	}).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := a.RunReport(); err != nil {
		t.Fatal(err)
	}
	var count int64
	a.DB().Model(&model.Snapshot{}).Count(&count)
	if count != 2 {
		t.Fatalf("RunReport must not create a reporting-time snapshot, snapshots = %d", count)
	}
	latest, err := db.LatestSnapshot(a.DB())
	if err != nil {
		t.Fatal(err)
	}
	if latest.AccountUsed != 80000000 {
		t.Fatalf("latest snapshot used = %d, want 80000000", latest.AccountUsed)
	}
	for _, want := range []string{
		"已用配额", "80000000", "+30000000",
		"当前余额", "$400.00", "当前余额（消耗）：+$20.00",
		"历史消耗", "$240.00", "+$60.00",
		"live-key",
		"可用总量", "$4.00", "可用总量（消耗）：+$1.00",
		"累计已用", "$3.00", "+$1.00",
		"授予总量", "$7.00", "+$2.00",
	} {
		if !strings.Contains(webhookBody, want) {
			t.Fatalf("report missing %q in body: %s", want, webhookBody)
		}
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
