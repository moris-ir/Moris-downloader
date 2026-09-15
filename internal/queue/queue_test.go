package queue

import (
	"context"
	"github.com/moris3245/moris/internal/domain"
	"testing"
	"time"
)

func TestQueue(t *testing.T) {
	q := New(1)
	j := domain.Job{ID: "1"}
	if err := q.Enqueue(context.Background(), j); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	got, err := q.Dequeue(ctx)
	if err != nil || got.ID != "1" {
		t.Fatalf("%v %#v", err, got)
	}
}
