package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetTokenUsage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token-key" {
			t.Errorf("usage must use token key as bearer, got %s", r.Header.Get("Authorization"))
		}
		w.Write([]byte(`{"data":{"name":"claude","total_used":888,"total_granted":1000,"unlimited_quota":true},"message":"","success":true}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "sys-token", "827947")
	d, _, err := c.GetTokenUsage("token-key")
	if err != nil {
		t.Fatal(err)
	}
	if d.TotalUsed != 888 {
		t.Errorf("total_used = %d", d.TotalUsed)
	}
}

func TestGetTokenUsageNoUserHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("new-api-user") != "" {
			t.Errorf("usage must NOT send new-api-user, got %s", r.Header.Get("new-api-user"))
		}
		if r.URL.Path != "/api/usage/token/" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write([]byte(`{"data":{"name":"x"},"success":true}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "sys-token", "827947")
	if _, _, err := c.GetTokenUsage("token-key"); err != nil {
		t.Fatal(err)
	}
}
