package cache

import (
	"sync"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/eko/gocache/lib/v4/cache"
	ristretto_store "github.com/eko/gocache/store/ristretto/v4"
)

func NewCache[T any]() cache.CacheInterface[T] {
	ristrettoCache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000,
		MaxCost:     100,
		BufferItems: 64,
	})
	if err != nil {
		panic(err)
	}

	ristrettoStore := ristretto_store.NewRistretto(ristrettoCache)

	return cache.New[T](ristrettoStore)
}

type SimpleExpirationCache[T any] struct {
	expiration  time.Duration
	lastUpdated time.Time
	mutex       sync.Mutex
	val         T
}

func NewSimpleExpirationCache[T any](expiration time.Duration) *SimpleExpirationCache[T] {
	return &SimpleExpirationCache[T]{expiration: expiration}
}

func (c *SimpleExpirationCache[T]) Get() (T, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.val, time.Since(c.lastUpdated) < c.expiration
}

func (c *SimpleExpirationCache[T]) Set(val T) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.val = val
	c.lastUpdated = time.Now()
}
