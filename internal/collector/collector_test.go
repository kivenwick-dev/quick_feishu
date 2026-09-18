package collector

import (
	"net/http"
	"net/http/httptest"
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
