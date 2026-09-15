package app

import (
	"context"
	"github.com/moris3245/moris/internal/config"
	"github.com/moris3245/moris/internal/domain"
	"testing"
)

func TestCancelAndRetryJob(t *testing.T) {
	a := New(config.Config{StorageDir: t.TempDir(), QueueMode: "memory"})
	j, err := a.CreateJob(context.Background(), "1", "https://example.com/x", domain.JobVideo, "mp4", "best")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.CancelJob(context.Background(), j.ID, "1"); err != nil {
		t.Fatal(err)
	}
	got, ok, _ := a.Repo.GetJob(context.Background(), j.ID)
	if !ok || got.Status != domain.JobCancelled {
		t.Fatalf("unexpected status: %+v", got)
	}
	got.Status = domain.JobFailed
	_ = a.Repo.UpdateJob(context.Background(), got)
	if _, err = a.RetryJob(context.Background(), j.ID); err != nil {
		t.Fatal(err)
	}
	got, _, _ = a.Repo.GetJob(context.Background(), j.ID)
	if got.Status != domain.JobQueued {
		t.Fatalf("expected queued, got %s", got.Status)
	}
}
