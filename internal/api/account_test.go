package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetAccount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user/self" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write([]byte(`{"data":{"id":827947,"quota":100,"used_quota":50,"request_count":10},"message":"","success":true}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "sys-token", "827947")
	d, raw, err := c.GetAccount()
	if err != nil {
		t.Fatal(err)
	}
	if d.Quota != 100 || d.UsedQuota != 50 {
		t.Errorf("bad data: %+v", d)
	}
	if len(raw) == 0 {
		t.Error("raw should not be empty")
	}
	if string(d.Raw) == "" {
		t.Error("data.Raw should be set")
	}
}
