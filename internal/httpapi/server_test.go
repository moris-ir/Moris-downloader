package httpapi

import (
	"bytes"
	"github.com/moris3245/moris/internal/app"
	"github.com/moris3245/moris/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthAndJobAPI(t *testing.T) {
	a := app.New(config.Config{StorageDir: t.TempDir(), MaxFileSizeMB: 10})
	s := httptest.NewServer(New(a).Handler())
	defer s.Close()
	r, err := http.Get(s.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != http.StatusOK {
		t.Fatalf("health status=%d", r.StatusCode)
	}
	body := bytes.NewBufferString(`{"userID":"u1","url":"https://example.com/a","type":"video"}`)
	r, err = http.Post(s.URL+"/api/v1/jobs", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != http.StatusOK {
		t.Fatalf("job status=%d", r.StatusCode)
	}
}
