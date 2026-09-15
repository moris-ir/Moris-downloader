package repository

import (
	"context"
	"github.com/moris3245/moris/internal/domain"
	"testing"
	"time"
)

func TestFileRepositoryPersistsAcrossInstances(t *testing.T) {
	dir := t.TempDir()
	a := NewFile(dir)
	b := NewFile(dir)
	j := domain.Job{ID: "job-1", UserID: "42", URL: "https://example.com", Status: domain.JobQueued, CreatedAt: time.Now().UTC()}
	if err := a.CreateJob(context.Background(), j); err != nil {
		t.Fatal(err)
	}
	got, ok, err := b.GetJob(context.Background(), j.ID)
	if err != nil || !ok {
		t.Fatalf("err=%v ok=%v", err, ok)
	}
	if got.ID != j.ID {
		t.Fatalf("got %#v", got)
	}
	got.Status = domain.JobCompleted
	if err = b.UpdateJob(context.Background(), got); err != nil {
		t.Fatal(err)
	}
	again, _, err := a.GetJob(context.Background(), j.ID)
	if err != nil {
		t.Fatal(err)
	}
	if again.Status != domain.JobCompleted {
		t.Fatalf("status=%s", again.Status)
	}
}
