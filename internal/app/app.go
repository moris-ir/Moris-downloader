package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/moris3245/moris/internal/config"
	"github.com/moris3245/moris/internal/domain"
	"github.com/moris3245/moris/internal/media"
	"github.com/moris3245/moris/internal/music"
	"github.com/moris3245/moris/internal/queue"
	"github.com/moris3245/moris/internal/redisx"
	"github.com/moris3245/moris/internal/repository"
	"github.com/moris3245/moris/internal/telegram"
	"strconv"
	"strings"
	"time"
)

type App struct {
	Cfg        config.Config
	Repo       repository.Repository
	Queue      queue.Queue
	Media      *media.Engine
	Telegram   *telegram.Client
	Redis      *redisx.Client
	Recognizer *music.AudD
	Lyrics     *music.LRCLIB
}

func New(cfg config.Config) *App {
	var q queue.Queue = queue.New(100)
	var r *redisx.Client
	if cfg.QueueMode == "redis" {
		r = redisx.New(cfg.RedisAddr)
		q = queue.NewRedis(cfg.RedisAddr, cfg.RedisQueueKey)
	}
	return &App{Cfg: cfg, Repo: repository.NewFile(cfg.StorageDir), Queue: q, Media: media.NewEngine(cfg.StorageDir, cfg.MaxFileSizeMB*1024*1024), Telegram: telegram.New(cfg.TelegramToken), Redis: r, Recognizer: music.NewAudD(cfg.AudDToken), Lyrics: music.NewLRCLIB()}
}
func NewID() string {
	b := make([]byte, 12)
	if _, e := rand.Read(b); e != nil {
		return fmt.Sprintf("job-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
func (a *App) ensureUser(ctx context.Context, id string) error {
	u, ok, e := a.Repo.GetUser(ctx, id)
	if e != nil {
		return e
	}
	now := time.Now().UTC()
	if !ok {
		u = domain.User{ID: id, Status: domain.UserActive, Plan: domain.PlanFree, RegisteredAt: now}
	}
	u.LastActivityAt = now
	if u.DefaultVideoQuality == "" {
		u.DefaultVideoQuality = "best"
	}
	if u.DefaultAudioQuality == "" {
		u.DefaultAudioQuality = "m4a"
	}
	if u.PreferredFormat == "" {
		u.PreferredFormat = "mp4"
	}
	if n, e := strconv.ParseInt(id, 10, 64); e == nil {
		u.TelegramID = n
	}
	return a.Repo.UpsertUser(ctx, u)
}
func (a *App) allowed(ctx context.Context, userID string) error {
	if a.Cfg.RateLimitPerMinute <= 0 {
		return nil
	}
	if a.Redis == nil {
		return nil
	}
	key := "moris:rate:" + userID + ":" + time.Now().UTC().Format("200601021504")
	v, e := a.Redis.Do(ctx, "INCR", key)
	if e != nil {
		return nil
	}
	n, ok := v.(int64)
	if ok && n == 1 {
		_, _ = a.Redis.Do(ctx, "EXPIRE", key, "61")
	}
	if ok && n > int64(a.Cfg.RateLimitPerMinute) {
		return errors.New("rate limit exceeded; please try again later")
	}
	return nil
}
func (a *App) CreateJob(ctx context.Context, userID, url string, jt domain.JobType, format, quality string) (domain.Job, error) {
	if userID == "" {
		return domain.Job{}, errors.New("user id is required")
	}
	if err := a.ensureUser(ctx, userID); err != nil {
		return domain.Job{}, err
	}
	if err := a.allowed(ctx, userID); err != nil {
		return domain.Job{}, err
	}
	u, _, _ := a.Repo.GetUser(ctx, userID)
	if u.Status == domain.UserBanned {
		return domain.Job{}, errors.New("user is banned")
	}
	if jt == domain.JobVideo && format == "" {
		format = u.PreferredFormat
	}
	if jt == domain.JobAudio && format == "" {
		format = u.DefaultAudioQuality
	}
	if jt == domain.JobVideo && quality == "" {
		quality = u.DefaultVideoQuality
	}
	j := domain.Job{ID: NewID(), UserID: userID, URL: strings.TrimSpace(url), Type: jt, Format: strings.ToLower(strings.TrimSpace(format)), Quality: strings.ToLower(strings.TrimSpace(quality)), Status: domain.JobPending, CreatedAt: time.Now().UTC()}
	if n, e := strconv.ParseInt(userID, 10, 64); e == nil {
		j.TelegramChatID = n
	}
	if e := media.ValidateURL(j.URL); e != nil {
		j.Status = domain.JobFailed
		j.Error = e.Error()
		_ = a.Repo.CreateJob(ctx, j)
		return j, e
	}
	if e := a.Repo.CreateJob(ctx, j); e != nil {
		return j, e
	}
	j.Status = domain.JobQueued
	if e := a.Repo.UpdateJob(ctx, j); e != nil {
		return j, e
	}
	if e := a.Queue.Enqueue(ctx, j); e != nil {
		j.Status = domain.JobFailed
		j.Error = e.Error()
		_ = a.Repo.UpdateJob(ctx, j)
		return j, e
	}
	return j, nil
}
func (a *App) RunWorkers(ctx context.Context) {
	for i := 0; i < a.Cfg.WorkerConcurrency; i++ {
		go a.worker(ctx)
	}
	go a.cleanupLoop(ctx)
}
func (a *App) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	_ = a.Media.Cleanup(a.Cfg.CleanupAge)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = a.Media.Cleanup(a.Cfg.CleanupAge)
		}
	}
}
func (a *App) worker(ctx context.Context) {
	for {
		j, e := a.Queue.Dequeue(ctx)
		if e != nil {
			return
		}
		if j.Status == domain.JobCancelled {
			continue
		}
		j.Status = domain.JobProcessing
		j.StartedAt = time.Now().UTC()
		j.Attempts++
		_ = a.Repo.UpdateJob(ctx, j)
		jobCtx, cancel := context.WithTimeout(ctx, a.Cfg.JobTimeout)
		result, e := a.Media.Download(jobCtx, j)
		cancel()
		if e != nil {
			if j.Attempts < a.Cfg.MaxAttempts && ctx.Err() == nil {
				j.Status = domain.JobQueued
				j.Error = "retry: " + e.Error()
				_ = a.Repo.UpdateJob(ctx, j)
				delay := a.Cfg.RetryBase * time.Duration(j.Attempts)
				t := time.NewTimer(delay)
				select {
				case <-ctx.Done():
					t.Stop()
					return
				case <-t.C:
				}
				_ = a.Queue.Enqueue(ctx, j)
				continue
			}
			j.Status = domain.JobFailed
			j.Error = e.Error()
			j.FinishedAt = time.Now().UTC()
			_ = a.Repo.UpdateJob(ctx, j)
			continue
		}
		j.OutputPath = result.Path
		j.OutputSize = result.Size
		if j.TelegramChatID > 0 && a.Cfg.TelegramToken != "" {
			if result.Size > a.Cfg.TelegramMaxUploadMB*1024*1024 {
				j.Status = domain.JobFailed
				j.Error = "output exceeds Telegram upload limit"
			} else {
				j.Status = domain.JobUploading
				_ = a.Repo.UpdateJob(ctx, j)
				field := "document"
				if j.Type == domain.JobVideo {
					field = "video"
				} else if j.Type == domain.JobAudio {
					field = "audio"
				}
				if e := a.Telegram.SendFile(ctx, j.TelegramChatID, result.Path, field, "MORIS • "+j.ID); e != nil {
					j.Status = domain.JobFailed
					j.Error = "telegram upload: " + e.Error()
				} else {
					j.Status = domain.JobCompleted
				}
			}
		} else {
			j.Status = domain.JobCompleted
		}
		j.FinishedAt = time.Now().UTC()
		_ = a.Repo.UpdateJob(ctx, j)
		if u, ok, _ := a.Repo.GetUser(ctx, j.UserID); ok {
			if j.Status == domain.JobCompleted {
				u.DownloadCount++
				u.StorageUsage += j.OutputSize
			} else if j.Status == domain.JobFailed {
				u.FailedJobs++
			}
			_ = a.Repo.UpsertUser(ctx, u)
		}
	}
}

func (a *App) CancelJob(ctx context.Context, id, userID string) (domain.Job, error) {
	j, ok, err := a.Repo.GetJob(ctx, id)
	if err != nil {
		return domain.Job{}, err
	}
	if !ok {
		return domain.Job{}, errors.New("job not found")
	}
	if userID != "" && j.UserID != userID {
		return domain.Job{}, errors.New("forbidden")
	}
	if j.Status == domain.JobCompleted || j.Status == domain.JobFailed {
		return j, errors.New("job already finished")
	}
	j.Status = domain.JobCancelled
	j.Error = "cancelled by user"
	j.FinishedAt = time.Now().UTC()
	if err := a.Repo.UpdateJob(ctx, j); err != nil {
		return domain.Job{}, err
	}
	return j, nil
}

func (a *App) RetryJob(ctx context.Context, id string) (domain.Job, error) {
	j, ok, err := a.Repo.GetJob(ctx, id)
	if err != nil {
		return domain.Job{}, err
	}
	if !ok {
		return domain.Job{}, errors.New("job not found")
	}
	if j.Status != domain.JobFailed && j.Status != domain.JobCancelled {
		return j, errors.New("job is not retryable")
	}
	j.Status, j.Error, j.FinishedAt = domain.JobQueued, "", time.Time{}
	if err := a.Repo.UpdateJob(ctx, j); err != nil {
		return domain.Job{}, err
	}
	if err := a.Queue.Enqueue(ctx, j); err != nil {
		return domain.Job{}, err
	}
	return j, nil
}

func (a *App) UpdateUserSettings(ctx context.Context, u domain.User) error {
	u.LastActivityAt = time.Now().UTC()
	return a.Repo.UpsertUser(ctx, u)
}
