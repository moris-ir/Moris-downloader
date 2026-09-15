package queue

import (
	"context"
	"errors"
	"github.com/moris3245/moris/internal/domain"
	"sync"
)

var ErrClosed = errors.New("queue closed")

type Queue interface {
	Enqueue(context.Context, domain.Job) error
	Dequeue(context.Context) (domain.Job, error)
	Len(context.Context) (int, error)
}

type Memory struct {
	ch     chan domain.Job
	mu     sync.RWMutex
	closed bool
}

func New(size int) *Memory {
	if size < 1 {
		size = 100
	}
	return &Memory{ch: make(chan domain.Job, size)}
}
func (q *Memory) Enqueue(ctx context.Context, j domain.Job) error {
	q.mu.RLock()
	defer q.mu.RUnlock()
	if q.closed {
		return ErrClosed
	}
	select {
	case q.ch <- j:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (q *Memory) Dequeue(ctx context.Context) (domain.Job, error) {
	select {
	case j := <-q.ch:
		return j, nil
	case <-ctx.Done():
		return domain.Job{}, ctx.Err()
	}
}
func (q *Memory) Len(_ context.Context) (int, error) { return len(q.ch), nil }
func (q *Memory) Close()                             { q.mu.Lock(); q.closed = true; q.mu.Unlock() }
