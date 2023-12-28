package cache

import (
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
