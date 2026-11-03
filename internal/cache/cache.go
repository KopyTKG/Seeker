package cache

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("cache: key not found")

type Cache interface {
	Close()
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, data string) error
	Exists(ctx context.Context, key string) (bool, error)
}
