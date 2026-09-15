package httpapi

import (
	"context"
	"encoding/json"
	"github.com/moris3245/moris/internal/app"
	"github.com/moris3245/moris/internal/domain"
	"github.com/moris3245/moris/internal/media"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Server struct{ App *app.App }

func New(a *app.App) *Server { return &Server{App: a} }
func (s *Server) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/admin", s.adminPage)
	m.HandleFunc("/health", s.health)
	m.HandleFunc("/ready", s.ready)
	m.HandleFunc("/metrics", s.metrics)
	m.HandleFunc("/api/v1/media/analyze", s.analyze)
	m.HandleFunc("/api/v1/music/recognize", s.recognize)
	m.HandleFunc("/api/v1/music/lyrics", s.lyrics)
	m.HandleFunc("/api/v1/jobs", s.jobs)
	m.HandleFunc("/api/v1/jobs/", s.job)
	m.HandleFunc("/api/v1/users/", s.user)
	m.HandleFunc("/api/v1/admin/stats", s.adminStats)
	m.HandleFunc("/api/v1/admin/jobs", s.adminJobs)
	m.HandleFunc("/api/v1/admin/users", s.adminUsers)
	return m
}
func (s *Server) adminPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>MORIS Admin</title><style>body{margin:0;background:#090b10;color:#eee;font:14px system-ui}main{max-width:1200px;margin:auto;padding:28px}.grid{display:grid;grid-template-columns:repeat(4,1fr);gap:12px}.card{background:#121722;border:1px solid #252c3a;border-radius:14px;padding:18px}.muted{color:#8f9aae}input,button{background:#0d1119;color:#fff;border:1px solid #30394a;border-radius:8px;padding:10px}button{cursor:pointer}pre{white-space:pre-wrap;overflow:auto}.row{display:flex;gap:8px;margin:12px 0}@media(max-width:800px){.grid{grid-template-columns:1fr 1fr}}</style></head><body><main><h1>MORIS Control Center</h1><p class="muted">Enter the admin token to inspect live service state.</p><div class="row"><input id="token" type="password" placeholder="Admin token" style="flex:1"><button onclick="load()">Refresh</button></div><section class="grid" id="stats"></section><div class="card" style="margin-top:16px"><h2>Recent Jobs</h2><pre id="jobs">—</pre></div><div class="card" style="margin-top:16px"><h2>Users</h2><pre id="users">—</pre></div></main><script>async function get(p){let t=document.getElementById('token').value;let r=await fetch(p,{headers:{'X-MORIS-ADMIN-TOKEN':t}});if(!r.ok)throw new Error(r.status);return r.json()}async function load(){try{let [s,j,u]=await Promise.all([get('/api/v1/admin/stats'),get('/api/v1/admin/jobs?limit=30'),get('/api/v1/admin/users?limit=30')]);document.getElementById('stats').innerHTML=Object.entries(s).map(([k,v])=>'<div class="card"><div class="muted">'+k+'</div><div style="font-size:28px">'+v+'</div></div>').join('');document.getElementById('jobs').textContent=JSON.stringify(j,null,2);document.getElementById('users').textContent=JSON.stringify(u,null,2)}catch(e){alert('Unauthorized or unavailable')}} </script></body></html>`))
}

func (s *Server) recognize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || (!s.botAuth(r) && !s.adminAuth(r)) {
		http.Error(w, "unauthorized", 401)
		return
	}
	var in struct {
		URL string `json:"url"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&in) != nil {
		http.Error(w, "bad json", 400)
		return
	}
	j := domain.Job{ID: app.NewID(), UserID: "music-recognition", URL: strings.TrimSpace(in.URL), Type: domain.JobAudio, Format: "m4a", Quality: "best"}
	if err := media.ValidateURL(j.URL); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	// Recognition is intentionally synchronous and bounded; it is an auxiliary intelligence endpoint, not the main queue.
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	res, e := s.App.Media.Download(ctx, j)
	if e != nil {
		http.Error(w, e.Error(), 502)
		return
	}
	defer os.RemoveAll(filepath.Dir(res.Path))
	track, e := s.App.Recognizer.Recognize(ctx, res.Path)
	if e != nil {
		http.Error(w, e.Error(), 502)
		return
	}
	writeJSON(w, 200, track)
}
func (s *Server) lyrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || (!s.botAuth(r) && !s.adminAuth(r)) {
		http.Error(w, "unauthorized", 401)
		return
	}
	artist := r.URL.Query().Get("artist")
	title := r.URL.Query().Get("title")
	if artist == "" || title == "" {
		http.Error(w, "artist and title are required", 400)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	out, e := s.App.Lyrics.Search(ctx, artist, title)
	if e != nil {
		http.Error(w, e.Error(), 404)
		return
	}
	writeJSON(w, 200, out)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	status := map[string]string{"status": "ready", "storage": "ok", "queue": "ok"}
	if _, e := s.App.Repo.Stats(r.Context()); e != nil {
		status["storage"] = "error"
		status["status"] = "degraded"
	}
	if s.App.Cfg.QueueMode == "redis" {
		if q, ok := s.App.Queue.(interface{ Ping(context.Context) error }); ok {
			if e := q.Ping(r.Context()); e != nil {
				status["queue"] = "error"
				status["status"] = "degraded"
			}
		}
	}
	writeJSON(w, 200, status)
}
func (s *Server) metrics(w http.ResponseWriter, r *http.Request) {
	stats, _ := s.App.Repo.Stats(r.Context())
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	for k, v := range stats {
		w.Write([]byte("moris_" + k + " " + itoa(v) + "\n"))
	}
	n, _ := s.App.Queue.Len(r.Context())
	w.Write([]byte("moris_queue_size " + itoa(int64(n)) + "\n"))
}
func (s *Server) botAuth(r *http.Request) bool {
	return s.App.Cfg.AdminToken == "" || r.Header.Get("X-MORIS-BOT-TOKEN") == s.App.Cfg.AdminToken
}
func (s *Server) adminAuth(r *http.Request) bool {
	return s.App.Cfg.AdminToken != "" && r.Header.Get("X-MORIS-ADMIN-TOKEN") == s.App.Cfg.AdminToken
}
func (s *Server) analyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	if !s.botAuth(r) && !s.adminAuth(r) {
		http.Error(w, "unauthorized", 401)
		return
	}
	var in struct {
		URL string `json:"url"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&in) != nil {
		http.Error(w, "bad json", 400)
		return
	}
	mi, e := s.App.Media.Analyze(r.Context(), in.URL)
	if e != nil {
		http.Error(w, e.Error(), 400)
		return
	}
	writeJSON(w, 200, mi)
}
func (s *Server) jobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	if !s.botAuth(r) && !s.adminAuth(r) {
		http.Error(w, "unauthorized", 401)
		return
	}
	var in struct {
		UserID       string `json:"user_id"`
		UserIDLegacy string `json:"userID"`
		URL          string `json:"url"`
		Format       string `json:"format"`
		Quality      string `json:"quality"`
		Type         string `json:"type"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&in) != nil {
		http.Error(w, "bad json", 400)
		return
	}
	uid := in.UserID
	if uid == "" {
		uid = in.UserIDLegacy
	}
	jt := domain.JobVideo
	if strings.EqualFold(in.Type, "audio") {
		jt = domain.JobAudio
	}
	j, e := s.App.CreateJob(r.Context(), uid, in.URL, jt, in.Format, in.Quality)
	if e != nil {
		http.Error(w, e.Error(), 400)
		return
	}
	writeJSON(w, 200, j)
}
func (s *Server) job(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/jobs/")
	parts := strings.Split(strings.Trim(id, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	jid := parts[0]
	switch r.Method {
	case http.MethodGet:
		if !s.botAuth(r) && !s.adminAuth(r) {
			http.Error(w, "unauthorized", 401)
			return
		}
		j, ok, e := s.App.Repo.GetJob(r.Context(), jid)
		if e != nil {
			http.Error(w, e.Error(), 500)
			return
		}
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, 200, j)
	case http.MethodPost:
		if !s.botAuth(r) && !s.adminAuth(r) {
			http.Error(w, "unauthorized", 401)
			return
		}
		if len(parts) < 2 {
			http.Error(w, "action required", 400)
			return
		}
		action := parts[1]
		if action == "cancel" {
			j, e := s.App.CancelJob(r.Context(), jid, r.URL.Query().Get("user_id"))
			if e != nil {
				http.Error(w, e.Error(), 400)
				return
			}
			writeJSON(w, 200, j)
			return
		}
		if action == "retry" {
			if !s.adminAuth(r) {
				http.Error(w, "admin only", 403)
				return
			}
			j, e := s.App.RetryJob(r.Context(), jid)
			if e != nil {
				http.Error(w, e.Error(), 400)
				return
			}
			writeJSON(w, 200, j)
			return
		}
		http.NotFound(w, r)
	default:
		http.Error(w, "method not allowed", 405)
	}
}
func (s *Server) user(w http.ResponseWriter, r *http.Request) {
	if !s.botAuth(r) && !s.adminAuth(r) {
		http.Error(w, "unauthorized", 401)
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/users/"), "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	u, ok, e := s.App.Repo.GetUser(r.Context(), id)
	if e != nil {
		http.Error(w, e.Error(), 500)
		return
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodGet {
		if r.URL.Query().Get("history") == "1" {
			jobs, e := s.App.Repo.ListJobsByUser(r.Context(), id, 100)
			if e != nil {
				http.Error(w, e.Error(), 500)
				return
			}
			writeJSON(w, 200, jobs)
			return
		}
		writeJSON(w, 200, u)
		return
	}
	if r.Method == http.MethodPut {
		var in struct {
			DefaultVideoQuality string `json:"default_video_quality"`
			DefaultAudioQuality string `json:"default_audio_quality"`
			PreferredFormat     string `json:"preferred_format"`
			Notifications       *bool  `json:"notifications"`
			AutoDownload        *bool  `json:"auto_download"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&in) != nil {
			http.Error(w, "bad json", 400)
			return
		}
		if in.DefaultVideoQuality != "" {
			u.DefaultVideoQuality = in.DefaultVideoQuality
		}
		if in.DefaultAudioQuality != "" {
			u.DefaultAudioQuality = in.DefaultAudioQuality
		}
		if in.PreferredFormat != "" {
			u.PreferredFormat = in.PreferredFormat
		}
		if in.Notifications != nil {
			u.Notifications = *in.Notifications
		}
		if in.AutoDownload != nil {
			u.AutoDownload = *in.AutoDownload
		}
		if e := s.App.UpdateUserSettings(r.Context(), u); e != nil {
			http.Error(w, e.Error(), 500)
			return
		}
		writeJSON(w, 200, u)
		return
	}
	http.Error(w, "method not allowed", 405)
}
func (s *Server) adminStats(w http.ResponseWriter, r *http.Request) {
	if !s.adminAuth(r) {
		http.Error(w, "unauthorized", 401)
		return
	}
	stats, e := s.App.Repo.Stats(r.Context())
	if e != nil {
		http.Error(w, e.Error(), 500)
		return
	}
	writeJSON(w, 200, stats)
}
func (s *Server) adminJobs(w http.ResponseWriter, r *http.Request) {
	if !s.adminAuth(r) {
		http.Error(w, "unauthorized", 401)
		return
	}
	n := 100
	if x, e := strconv.Atoi(r.URL.Query().Get("limit")); e == nil && x > 0 && x <= 1000 {
		n = x
	}
	jobs, e := s.App.Repo.ListJobs(r.Context(), n)
	if e != nil {
		http.Error(w, e.Error(), 500)
		return
	}
	writeJSON(w, 200, jobs)
}
func (s *Server) adminUsers(w http.ResponseWriter, r *http.Request) {
	if !s.adminAuth(r) {
		http.Error(w, "unauthorized", 401)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	n := 100
	if x, e := strconv.Atoi(r.URL.Query().Get("limit")); e == nil && x > 0 && x <= 1000 {
		n = x
	}
	users, e := s.App.Repo.ListUsers(r.Context(), n)
	if e != nil {
		http.Error(w, e.Error(), 500)
		return
	}
	writeJSON(w, 200, users)
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	b := make([]byte, 0, 20)
	for v > 0 {
		b = append(b, byte('0'+v%10))
		v /= 10
	}
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}
