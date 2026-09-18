package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetTokenList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":{"page":1,"page_size":2,"total":2,"items":[
			{"id":1,"key":"k1","name":"claude","used_quota":10,"group":"G1"},
			{"id":2,"key":"k2","name":"gpt","used_quota":20,"group":"G2"}
		]},"message":"","success":true}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "sys-token", "827947")
	d, _, err := c.GetTokenList()
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Items) != 2 || d.Items[0].Key != "k1" {
		t.Errorf("bad items: %+v", d.Items)
	}
}

func TestGetTokenListPagination(t *testing.T) {
	var requestedP []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/token/" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("new-api-user") != "827947" {
			t.Errorf("missing new-api-user header")
		}
		if r.Header.Get("Authorization") != "Bearer sys-token" {
			t.Errorf("missing Authorization header")
		}
		if r.URL.Query().Get("size") != "100" {
			t.Errorf("size = %s", r.URL.Query().Get("size"))
		}
		p := r.URL.Query().Get("p")
		requestedP = append(requestedP, p)
		if p == "0" {
			w.Write([]byte(`{"data":{"page":1,"page_size":2,"total":3,"items":[
				{"id":1,"key":"k1","name":"a"},{"id":2,"key":"k2","name":"b"}
			]},"success":true}`))
		} else {
			w.Write([]byte(`{"data":{"page":2,"page_size":2,"total":3,"items":[
				{"id":3,"key":"k3","name":"c"}
			]},"success":true}`))
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "sys-token", "827947")
	d, _, err := c.GetTokenList()
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Items) != 3 {
		t.Errorf("items = %d, want 3", len(d.Items))
	}
	if len(requestedP) != 2 || requestedP[0] != "0" || requestedP[1] != "1" {
		t.Errorf("requested pages = %v, want [0 1]", requestedP)
	}
}
