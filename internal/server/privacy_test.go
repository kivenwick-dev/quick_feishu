package server

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"gorm.io/datatypes"
	"quick-feishu/internal/collector"
	"quick-feishu/internal/config"
	"quick-feishu/internal/db"
	"quick-feishu/internal/model"
	"quick-feishu/internal/report"
)

func TestReadAPIsNeverExposePrivateSnapshotData(t *testing.T) {
	h := newTestHandlers(t)
	h.Config = config.Default()
	h.Config.Account.UserID = "PRIVATE_ACCOUNT"
	h.Config.Account.SystemToken = "PRIVATE_SYSTEM_TOKEN"
	h.Config.Account.APIBase = "https://PRIVATE_HOST/?key=PRIVATE_KEY"
	h.Config.Feishu.WebhookURL = "https://PRIVATE_WEBHOOK"
	h.Config.ReportTemplate = mustToMap(&report.Template{Sections: []report.Section{
		{Name: "账号", Source: "account", Fields: []report.Field{{Field: "username"}, {Field: "quota", Diff: false}, {Field: "email"}}},
		{Name: "令牌", Source: "usage", PerToken: true, Fields: []report.Field{{Field: "key"}, {Field: "total_used", Diff: false}, {Field: "model_limits"}}},
	}})
	if err := db.SeedDicts(h.DB); err != nil {
		t.Fatal(err)
	}
	var latest model.Snapshot
	for i := 1; i <= 2; i++ {
		snap := model.Snapshot{SnapshotDate: fmt.Sprintf("2026-09-%02d", i), AccountQuota: int64(10 * i), AccountRaw: datatypes.JSON(fmt.Sprintf(`{"username":"PRIVATE_USERNAME","email":"PRIVATE_EMAIL","quota":%d,"unknown_future_field":"PRIVATE_FUTURE"}`, 10*i)), TokenListRaw: datatypes.JSON(`[{"key":"PRIVATE_KEY"}]`), TokenUsageRaw: datatypes.JSON(`{"nested":{"secret":"PRIVATE_RAW"}}`)}
		if err := h.DB.Create(&snap).Error; err != nil {
			t.Fatal(err)
		}
		tok := model.TokenSnapshot{SnapshotID: snap.ID, TokenID: 123, TokenName: "测试令牌名称", UsageRaw: datatypes.JSON(fmt.Sprintf(`{"total_used":%d,"model_limits":{"secret":"PRIVATE_MODEL"}}`, i*5)), ListRaw: datatypes.JSON(`{"key":"PRIVATE_TOKEN_KEY"}`)}
		if err := h.DB.Create(&tok).Error; err != nil {
			t.Fatal(err)
		}
		latest = snap
	}
	h.DB.Create(&model.SendLog{ErrorMsg: "PRIVATE_ERROR_URL", FeishuResp: "PRIVATE_RESPONSE"})
	s := New(0)
	s.RegisterRoutes(h)
	for _, path := range []string{"/api/dashboard", "/api/snapshots", fmt.Sprintf("/api/snapshots/%d", latest.ID), "/api/latest", "/api/history?source=account", "/api/history?source=usage&token_id=123", "/api/tokens", "/api/settings", "/api/sendlogs", "/api/compare?from=2026-09-01&to=2026-09-02"} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			s.Engine.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
			if w.Code != 200 {
				t.Fatalf("code = %d", w.Code)
			}
			body := w.Body.String()
			if strings.Contains(body, "PRIVATE_") {
				t.Fatalf("private sentinel leaked: %s", body)
			}
			for _, key := range []string{"account_raw", "token_list_raw", "token_usage_raw", "usage_raw", "list_raw", "feishu_resp"} {
				if strings.Contains(body, `"`+key+`"`) {
					t.Errorf("raw field leaked: %s", key)
				}
			}
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Error("missing no-store")
			}
			if strings.Contains(path, "/api/history") {
				var hist report.History
				if err := json.Unmarshal(w.Body.Bytes(), &hist); err != nil {
					t.Fatal(err)
				}
				for _, f := range hist.Fields {
					if !publicMetric(hist.Source, f.Path) {
						t.Errorf("private field %s", f.Path)
					}
				}
				key, want := "quota", "+10"
				if hist.Source == "usage" {
					key, want = "total_used", "+5"
				}
				if hist.Rows[1].Deltas[key] != want {
					t.Errorf("delta = %q", hist.Rows[1].Deltas[key])
				}
			}
			if path == "/api/latest" {
				var latest struct {
					Sections []report.SectionResult `json:"sections"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &latest); err != nil {
					t.Fatal(err)
				}
				if len(latest.Sections) != 2 {
					t.Fatalf("sections = %d", len(latest.Sections))
				}
				if got := latest.Sections[0].Fields[0]; !got.IsDiff || got.Delta != "+10" {
					t.Errorf("account delta missing: %+v", got)
				}
				if got := latest.Sections[1].Tokens[0].Metrics[0]; !got.HasDelta || got.Delta != "+5" {
					t.Errorf("token delta missing: %+v", got)
				}
			}
			if path == "/api/tokens" && !strings.Contains(body, `"token_name":"测试令牌名称"`) {
				t.Error("missing token name")
			}
		})
	}
	if h.Config.Account.SystemToken != "PRIVATE_SYSTEM_TOKEN" {
		t.Error("settings read mutated stored token")
	}
}

func TestPublicHistoryDropsNonNumericMetrics(t *testing.T) {
	h := publicHistory(&report.History{Source: "account", Fields: []report.HistoryField{{Path: "quota"}, {Path: "email"}}, Rows: []report.HistoryRow{{Values: map[string]string{"quota": "PRIVATE_VALUE", "email": "PRIVATE_EMAIL"}, Deltas: map[string]string{"quota": "PRIVATE_DELTA"}}}})
	if len(h.Rows[0].Values) != 0 || len(h.Rows[0].Deltas) != 0 {
		t.Fatal("non numeric metric exposed")
	}
}

func TestAllQuantitativeMetricsRemainVisible(t *testing.T) {
	fields := []string{"quota", "used_quota", "request_count", "aff_count", "aff_quota", "aff_history_quota", "top_up_rebate_count", "withdrawn_quota", "invoice_returned_quota", "support_ticket_cap"}
	history := &report.History{Source: "account", Rows: []report.HistoryRow{{Values: map[string]string{}, Deltas: map[string]string{}}}}
	for _, field := range fields {
		history.Fields = append(history.Fields, report.HistoryField{Path: field})
		history.Rows[0].Values[field] = "0"
		history.Rows[0].Deltas[field] = "-5"
	}
	got := publicHistory(history)
	if len(got.Fields) != len(fields) {
		t.Fatalf("fields = %d, want %d", len(got.Fields), len(fields))
	}
	for _, field := range fields {
		if got.Rows[0].Values[field] != "0" || got.Rows[0].Deltas[field] != "-5" {
			t.Errorf("lost metric %s", field)
		}
	}
	for _, field := range []string{"id", "inviter_id", "created_at", "role", "status", "password_version", "cost_price_marker", "email"} {
		if publicMetric("account", field) {
			t.Errorf("non-statistical field allowed: %s", field)
		}
	}
	tmpl := publicTemplate(&report.Template{Sections: []report.Section{{Source: "account", Fields: []report.Field{{Field: "quota"}}}}})
	if len(tmpl.Sections[0].Fields) != len(fields) {
		t.Fatal("dashboard did not append additional metrics")
	}
	for _, field := range tmpl.Sections[0].Fields {
		if !field.Diff {
			t.Errorf("missing diff: %s", field.Field)
		}
	}
}

func TestPublicIssuesKeepsTokenNameAndKind(t *testing.T) {
	out := publicIssues([]collector.Issue{
		{Scope: "usage", TokenName: "gpt6robodjo", Kind: collector.KindQuotaExhausted, Status: 401, Detail: "该令牌额度已用尽"},
	})
	if len(out) != 1 {
		t.Fatalf("view len = %d", len(out))
	}
	if out[0].TokenName != "gpt6robodjo" || out[0].Kind != collector.KindQuotaExhausted || out[0].Status != 401 {
		t.Fatalf("bad view: %+v", out[0])
	}
}
