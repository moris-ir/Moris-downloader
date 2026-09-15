package repository

import (
	"context"
	"github.com/moris3245/moris/internal/domain"
	"sync"
	"time"
)

type Repository interface {
	UpsertUser(context.Context, domain.User) error
	GetUser(context.Context, string) (domain.User, bool, error)
	ListUsers(context.Context, int) ([]domain.User, error)
	CreateJob(context.Context, domain.Job) error
	GetJob(context.Context, string) (domain.Job, bool, error)
	UpdateJob(context.Context, domain.Job) error
	ListJobs(context.Context, int) ([]domain.Job, error)
	ListJobsByUser(context.Context, string, int) ([]domain.Job, error)
	Stats(context.Context) (map[string]int64, error)
}

type Memory struct {
	mu    sync.RWMutex
	users map[string]domain.User
	jobs  map[string]domain.Job
}

func NewMemory() *Memory {
	return &Memory{users: map[string]domain.User{}, jobs: map[string]domain.Job{}}
}
func (m *Memory) UpsertUser(_ context.Context, u domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[u.ID] = u
	return nil
}
func (m *Memory) GetUser(_ context.Context, id string) (domain.User, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[id]
	return u, ok, nil
}
func (m *Memory) ListUsers(_ context.Context, n int) ([]domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]domain.User, 0, n)
	for _, u := range m.users {
		out = append(out, u)
		if len(out) >= n {
			break
		}
	}
	return out, nil
}
func (m *Memory) CreateJob(_ context.Context, j domain.Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[j.ID] = j
	return nil
}
func (m *Memory) GetJob(_ context.Context, id string) (domain.Job, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	j, ok := m.jobs[id]
	return j, ok, nil
}
func (m *Memory) UpdateJob(_ context.Context, j domain.Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[j.ID] = j
	return nil
}
func (m *Memory) ListJobs(_ context.Context, n int) ([]domain.Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]domain.Job, 0, n)
	for _, j := range m.jobs {
		out = append(out, j)
		if len(out) >= n {
			break
		}
	}
	return out, nil
}
func (m *Memory) ListJobsByUser(_ context.Context, userID string, n int) ([]domain.Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]domain.Job, 0, n)
	for _, j := range m.jobs {
		if j.UserID == userID {
			out = append(out, j)
			if len(out) >= n {
				break
			}
		}
	}
	return out, nil
}
func (m *Memory) Stats(_ context.Context) (map[string]int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s := map[string]int64{"users": int64(len(m.users)), "jobs": int64(len(m.jobs))}
	for _, j := range m.jobs {
		if j.Status == domain.JobCompleted {
			s["successful_downloads"]++
		}
		if j.Status == domain.JobFailed {
			s["failed_downloads"]++
		}
		if j.Status == domain.JobProcessing {
			s["processing_jobs"]++
		}
		if j.Status == domain.JobQueued {
			s["queued_jobs"]++
		}
	}
	return s, nil
}
func NewUser(tgID int64, username, first, lang string) domain.User {
	now := time.Now().UTC()
	return domain.User{ID: itoa(tgID), TelegramID: tgID, Username: username, FirstName: first, Language: lang, RegisteredAt: now, LastActivityAt: now, Status: domain.UserActive, Plan: domain.PlanFree}
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
