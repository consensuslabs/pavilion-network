package cache

import (
	"context"
	"time"
)

// Service defines the interface for cache operations
type Service interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
	Close() error
}
