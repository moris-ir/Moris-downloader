package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

type Local struct{ Root string }

func NewLocal(root string) (*Local, error) {
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, err
	}
	return &Local{Root: root}, nil
}
func (s *Local) Put(ctx context.Context, key string, r io.Reader) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	p := filepath.Join(s.Root, filepath.Clean("/"+key))
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return "", err
	}
	f, err := os.Create(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err = io.Copy(f, r); err != nil {
		return "", err
	}
	return p, nil
}
func (s *Local) Delete(_ context.Context, key string) error {
	return os.Remove(filepath.Join(s.Root, filepath.Clean("/"+key)))
}
