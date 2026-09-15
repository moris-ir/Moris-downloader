package httpapi

import (
	"bytes"
	"github.com/moris3245/moris/internal/app"
	"github.com/moris3245/moris/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJobAPIRequiresBotTokenWhenConfigured(t *testing.T) {
	a := app.New(config.Config{StorageDir: t.TempDir(), MaxFileSizeMB: 10, AdminToken: "secret"})
	srv := httptest.NewServer(New(a).Handler())
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/jobs", bytes.NewBufferString(`{"userID":"1","url":"https://example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	r, e := http.DefaultClient.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	r.Body.Close()
	if r.StatusCode != http.StatusUnauthorized {
		t.Fatalf("got %d", r.StatusCode)
	}
	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/api/v1/jobs", bytes.NewBufferString(`{"userID":"1","url":"https://example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-MORIS-BOT-TOKEN", "secret")
	r, e = http.DefaultClient.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	r.Body.Close()
	if r.StatusCode != http.StatusOK {
		t.Fatalf("got %d", r.StatusCode)
	}
}
