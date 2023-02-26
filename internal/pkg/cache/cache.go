package cache

import "context"

// Cache - interface for add, delete and check existence of rand values for hashcash
type Cache interface {
	// Add - add rand value with expiration (in seconds) to cache
	Add(ctx context.Context, key string, value int64) error
	// Get - check existence of int key in cache
	Exist(ctx context.Context, key string) (bool, error)
	// Delete - delete key from cache
	Delete(ctx context.Context, key string)
}
