package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/moris3245/moris/internal/domain"
	"github.com/moris3245/moris/internal/redisx"
	"strconv"
	"time"
)

type RedisQueue struct {
	c   *redisx.Client
	key string
}

func NewRedis(addr, key string) *RedisQueue { return &RedisQueue{c: redisx.New(addr), key: key} }
func (q *RedisQueue) Enqueue(ctx context.Context, j domain.Job) error {
	b, e := json.Marshal(j)
	if e != nil {
		return e
	}
	_, e = q.c.Do(ctx, "RPUSH", q.key, string(b))
	return e
}
func (q *RedisQueue) Dequeue(ctx context.Context) (domain.Job, error) {
	v, e := q.c.Do(ctx, "BLPOP", q.key, "0")
	if e != nil {
		return domain.Job{}, e
	}
	a, ok := v.([]any)
	if !ok || len(a) != 2 {
		return domain.Job{}, fmt.Errorf("redis queue: invalid response")
	}
	s, ok := a[1].(string)
	if !ok {
		return domain.Job{}, fmt.Errorf("redis queue: invalid payload")
	}
	var j domain.Job
	if e = json.Unmarshal([]byte(s), &j); e != nil {
		return domain.Job{}, e
	}
	return j, nil
}
func (q *RedisQueue) Len(ctx context.Context) (int, error) {
	v, e := q.c.Do(ctx, "LLEN", q.key)
	if e != nil {
		return 0, e
	}
	n, ok := v.(int64)
	if !ok {
		return 0, fmt.Errorf("redis queue: invalid length")
	}
	return int(n), nil
}
func (q *RedisQueue) Ping(ctx context.Context) error {
	v, e := q.c.Do(ctx, "PING")
	if e != nil {
		return e
	}
	if v != "PONG" {
		return fmt.Errorf("redis ping: %v", v)
	}
	return nil
}
func (q *RedisQueue) String() string { return q.key + "@redis" }

var _ = strconv.Itoa
var _ = time.Second
