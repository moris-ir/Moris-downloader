package httpapi

import (
	"github.com/moris3245/moris/internal/app"
	"github.com/moris3245/moris/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminStatsToken(t *testing.T) {
	a := app.New(config.Config{StorageDir: t.TempDir(), AdminToken: "admin"})
	s := httptest.NewServer(New(a).Handler())
	defer s.Close()
	r, _ := http.Get(s.URL + "/api/v1/admin/stats")
	if r.StatusCode != 401 {
		t.Fatalf("got %d", r.StatusCode)
	}
	req, _ := http.NewRequest("GET", s.URL+"/api/v1/admin/stats", nil)
	req.Header.Set("X-MORIS-ADMIN-TOKEN", "admin")
	r, _ = http.DefaultClient.Do(req)
	if r.StatusCode != 200 {
		t.Fatalf("got %d", r.StatusCode)
	}
}
