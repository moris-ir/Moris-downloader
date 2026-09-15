package repository

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/moris3245/moris/internal/domain"
	"os"
	"path/filepath"
	"syscall"
)

type fileState struct {
	Users map[string]domain.User `json:"users"`
	Jobs  map[string]domain.Job  `json:"jobs"`
}
type File struct{ path string }

func NewFile(dir string) *File { return &File{path: filepath.Join(dir, "state", "repository.json")} }
func (f *File) withLock(ctx context.Context, write bool, fn func(*fileState) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f.path), 0755); err != nil {
		return err
	}
	lock, err := os.OpenFile(f.path+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()
	how := syscall.LOCK_SH
	if write {
		how = syscall.LOCK_EX
	}
	if err = syscall.Flock(int(lock.Fd()), how); err != nil {
		return err
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	st := fileState{Users: map[string]domain.User{}, Jobs: map[string]domain.Job{}}
	if b, e := os.ReadFile(f.path); e == nil && len(b) > 0 {
		if e = json.Unmarshal(b, &st); e != nil {
			return e
		}
	} else if e != nil && !errors.Is(e, os.ErrNotExist) {
		return e
	}
	if st.Users == nil {
		st.Users = map[string]domain.User{}
	}
	if st.Jobs == nil {
		st.Jobs = map[string]domain.Job{}
	}
	if err := fn(&st); err != nil {
		return err
	}
	if !write {
		return nil
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp := f.path + ".tmp"
	if err = os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, f.path)
}
func (f *File) UpsertUser(ctx context.Context, u domain.User) error {
	return f.withLock(ctx, true, func(s *fileState) error { s.Users[u.ID] = u; return nil })
}
func (f *File) ListUsers(ctx context.Context, n int) ([]domain.User, error) {
	out := []domain.User{}
	err := f.withLock(ctx, false, func(s *fileState) error {
		for _, u := range s.Users {
			out = append(out, u)
			if len(out) >= n {
				break
			}
		}
		return nil
	})
	return out, err
}

func (f *File) GetUser(ctx context.Context, id string) (domain.User, bool, error) {
	var u domain.User
	var ok bool
	err := f.withLock(ctx, false, func(s *fileState) error { u, ok = s.Users[id]; return nil })
	return u, ok, err
}
func (f *File) CreateJob(ctx context.Context, j domain.Job) error {
	return f.withLock(ctx, true, func(s *fileState) error { s.Jobs[j.ID] = j; return nil })
}
func (f *File) GetJob(ctx context.Context, id string) (domain.Job, bool, error) {
	var j domain.Job
	var ok bool
	err := f.withLock(ctx, false, func(s *fileState) error { j, ok = s.Jobs[id]; return nil })
	return j, ok, err
}
func (f *File) UpdateJob(ctx context.Context, j domain.Job) error {
	return f.withLock(ctx, true, func(s *fileState) error { s.Jobs[j.ID] = j; return nil })
}
func (f *File) ListJobs(ctx context.Context, n int) ([]domain.Job, error) {
	out := []domain.Job{}
	err := f.withLock(ctx, false, func(s *fileState) error {
		for _, j := range s.Jobs {
			out = append(out, j)
			if len(out) >= n {
				break
			}
		}
		return nil
	})
	return out, err
}
func (f *File) ListJobsByUser(ctx context.Context, userID string, n int) ([]domain.Job, error) {
	out := []domain.Job{}
	err := f.withLock(ctx, false, func(s *fileState) error {
		for _, j := range s.Jobs {
			if j.UserID == userID {
				out = append(out, j)
				if len(out) >= n {
					break
				}
			}
		}
		return nil
	})
	return out, err
}

func (f *File) Stats(ctx context.Context) (map[string]int64, error) {
	out := map[string]int64{}
	err := f.withLock(ctx, false, func(s *fileState) error {
		out["users"] = int64(len(s.Users))
		out["jobs"] = int64(len(s.Jobs))
		for _, j := range s.Jobs {
			if j.Status == domain.JobCompleted {
				out["successful_downloads"]++
			}
			if j.Status == domain.JobFailed {
				out["failed_downloads"]++
			}
			if j.Status == domain.JobProcessing {
				out["processing_jobs"]++
			}
			if j.Status == domain.JobQueued {
				out["queued_jobs"]++
			}
		}
		return nil
	})
	return out, err
}
