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
