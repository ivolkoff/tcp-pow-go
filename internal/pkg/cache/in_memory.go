package cache

import (
	"context"
	"sync"

	"github.com/ivolkoff/tcp-pow-go/internal/pkg/clock"
)

// InMemoryCache - implementation of Cache interface
// local in-memory storage, replacement for Redis in tests
// Mutex is used to protect map (sync.Map can be used too)
type InMemoryCache struct {
	dataMap map[string]inMemoryValue
	lock    *sync.Mutex
	clock   clock.Clock
}

// inMemoryValue - internal struct to check expiration on values in cache
type inMemoryValue struct {
	SetTime    int64
	Expiration int64
}

// InitInMemoryCache - create new instance of InMemoryCache
// clock - instance of Clock to get time.Now() (and mocks in tests)
func InitInMemoryCache(clock clock.Clock) *InMemoryCache {
	return &InMemoryCache{
		dataMap: make(map[string]inMemoryValue, 0),
		lock:    &sync.Mutex{},
		clock:   clock,
	}
}

// Add - add rand value with expiration (in seconds) to cache
func (c *InMemoryCache) Add(ctx context.Context, key string, expiration int64) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.dataMap[key] = inMemoryValue{
		SetTime:    c.clock.Now().Unix(),
		Expiration: expiration,
	}
	return nil
}

// Exist - check existence of int key in cache
func (c *InMemoryCache) Exist(ctx context.Context, key string) (bool, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	value, ok := c.dataMap[key]
	if ok && c.clock.Now().Unix()-value.SetTime > value.Expiration {
		return false, nil
	}
	return ok, nil
}

// Delete - delete key from cache
func (c *InMemoryCache) Delete(ctx context.Context, key string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.dataMap, key)
}
