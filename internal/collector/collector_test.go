package collector

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"quick-feishu/internal/api"
	"quick-feishu/internal/db"
	"quick-feishu/internal/model"
)

func TestCollectAndSave(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/self":
			w.Write([]byte(`{"data":{"id":1,"quota":1000,"used_quota":300,"request_count":5},"success":true}`))
		case "/api/token/":
			w.Write([]byte(`{"data":{"page":1,"page_size":2,"total":2,"items":[
				{"id":11,"key":"k1","name":"claude","used_quota":100,"remain_quota":0},
				{"id":12,"key":"k2","name":"gpt","used_quota":200,"remain_quota":0}
			]},"success":true}`))
		case "/api/usage/token/":
			w.Write([]byte(`{"data":{"name":"x","total_used":100,"total_granted":1000},"success":true}`))
		}
	}))
	defer srv.Close()

	c := api.NewClient(srv.URL, "sys-token", "1")
	res := Collect(c)
	if len(res.Usages) != 2 {
		t.Errorf("usages = %d, want 2", len(res.Usages))
	}
	if res.Account == nil || res.Account.Quota != 1000 {
		t.Errorf("account bad: %+v", res.Account)
	}

	dir := t.TempDir()
	gdb, err := db.Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := Save(gdb, "2026-09-18", res); err != nil {
		t.Fatal(err)
	}
	var snap model.Snapshot
	gdb.First(&snap)
	if snap.AccountUsed != 300 {
		t.Errorf("snapshot used = %d", snap.AccountUsed)
	}
	if string(snap.AccountRaw) == "" || string(snap.TokenListRaw) == "" || string(snap.TokenUsageRaw) == "" {
		t.Error("raw JSON should be stored")
	}
	var tokens []model.TokenSnapshot
	gdb.Where("snapshot_id = ?", snap.ID).Find(&tokens)
	if len(tokens) != 2 {
		t.Errorf("tokens = %d, want 2", len(tokens))
	}
	if tokens[0].TotalUsed == 0 {
		t.Error("token TotalUsed should be populated from usage")
	}
}

func TestSaveStoresUnknownFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/self":
			w.Write([]byte(`{"data":{"id":1,"quota":1000,"used_quota":300,"request_count":5,"aff_code":"XYZ"},"success":true}`))
		case "/api/token/":
			w.Write([]byte(`{"data":{"page":1,"page_size":1,"total":1,"items":[
				{"id":11,"key":"k1","name":"claude","used_quota":100,"remain_quota":0,"mj_image_mode":"relax"}
			]},"success":true}`))
		case "/api/usage/token/":
			w.Write([]byte(`{"data":{"name":"x","total_used":100,"total_granted":1000,"extra_usage_field":"present"},"success":true}`))
		}
	}))
	defer srv.Close()

	c := api.NewClient(srv.URL, "sys-token", "1")
	res := Collect(c)
	if len(res.Errors) != 0 {
		t.Fatalf("collect errors: %v", res.Errors)
	}

	dir := t.TempDir()
	gdb, err := db.Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := Save(gdb, "2026-09-18", res); err != nil {
		t.Fatal(err)
	}

	var snap model.Snapshot
	gdb.First(&snap)
	if !strings.Contains(string(snap.AccountRaw), "aff_code") || !strings.Contains(string(snap.AccountRaw), "XYZ") {
		t.Errorf("account raw missing unknown field: %s", string(snap.AccountRaw))
	}
	if !strings.Contains(string(snap.TokenListRaw), "mj_image_mode") {
		t.Errorf("token list raw missing unknown token field: %s", string(snap.TokenListRaw))
	}

	var tokens []model.TokenSnapshot
	gdb.Where("snapshot_id = ?", snap.ID).Find(&tokens)
	if len(tokens) != 1 {
		t.Fatalf("tokens = %d, want 1", len(tokens))
	}
	if !strings.Contains(string(tokens[0].ListRaw), "mj_image_mode") {
		t.Errorf("token ListRaw missing unknown field: %s", string(tokens[0].ListRaw))
	}
	if !strings.Contains(string(tokens[0].UsageRaw), "extra_usage_field") {
		t.Errorf("token UsageRaw missing unknown field: %s", string(tokens[0].UsageRaw))
	}
}

func TestSaveUpsertsSameDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/self":
			w.Write([]byte(`{"data":{"id":9,"quota":1000,"used_quota":300,"request_count":5},"success":true}`))
		case "/api/token/":
			w.Write([]byte(`{"data":{"page":1,"page_size":2,"total":2,"items":[
				{"id":11,"key":"k1","name":"claude","used_quota":100},
				{"id":12,"key":"k2","name":"gpt","used_quota":200}
			]},"success":true}`))
		case "/api/usage/token/":
			w.Write([]byte(`{"data":{"name":"x","total_used":100},"success":true}`))
		}
	}))
	defer srv.Close()

	gdb, err := db.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	c := api.NewClient(srv.URL, "sys", "9")
	res := Collect(c)

	if err := Save(gdb, "2026-09-18", res); err != nil {
		t.Fatal(err)
	}
	if err := Save(gdb, "2026-09-18", res); err != nil {
		t.Fatal(err)
	}

	var snapCount int64
	gdb.Model(&model.Snapshot{}).Count(&snapCount)
	if snapCount != 1 {
		t.Errorf("snapshots = %d, want 1 (same date should upsert)", snapCount)
	}
	var tokCount int64
	gdb.Model(&model.TokenSnapshot{}).Count(&tokCount)
	if tokCount != 2 {
		t.Errorf("token snapshots = %d, want 2 (no orphans)", tokCount)
	}

	// 另一天应新增而非覆盖
	if err := Save(gdb, "2026-09-19", res); err != nil {
		t.Fatal(err)
	}
	gdb.Model(&model.Snapshot{}).Count(&snapCount)
	if snapCount != 2 {
		t.Errorf("snapshots = %d, want 2 after new date", snapCount)
	}
}
