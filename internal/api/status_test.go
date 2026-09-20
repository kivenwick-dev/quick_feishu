package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/status" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"quota_per_unit":500000,"display_in_currency":true},"success":true}`))
	}))
	defer srv.Close()

	status, err := NewClient(srv.URL, "", "").GetStatus()
	if err != nil {
		t.Fatal(err)
	}
	if status.QuotaPerUnit != 500000 || !status.DisplayInCurrency {
		t.Fatalf("status = %+v", status)
	}
}
