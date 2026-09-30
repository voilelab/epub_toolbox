package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestImage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte("img"))
	}))
	defer srv.Close()

	b, err := Image(context.Background(), srv.URL+"/a.png")
	if err != nil || string(b) != "img" {
		t.Errorf("Image = %q, %v", b, err)
	}
	if _, err := Image(context.Background(), srv.URL+"/missing"); err == nil {
		t.Error("want error for 404")
	}
}
