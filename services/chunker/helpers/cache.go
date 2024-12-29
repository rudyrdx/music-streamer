package helpers

import (
	"errors"
	"time"

	"github.com/patrickmn/go-cache"
)

func LookupFromCacheOrDB[T any](
	c *cache.Cache,
	key string,
	dbLookupFunc func() (T, error),
	expiration time.Duration,
) (T, error) {
	var zeroValue T // Default zero value for type T

	// Attempt to fetch from cache
	if cachedValue, found := c.Get(key); found {
		if typedValue, ok := cachedValue.(T); ok {
			return typedValue, nil // Return the value if it matches type T
		}
		return zeroValue, errors.New("type assertion failed for cached value")
	}

	// Fetch from database or other source if not in cache
	dbValue, err := dbLookupFunc()
	if err != nil {
		return zeroValue, err
	}

	// Store the fetched value in cache
	c.Set(key, dbValue, expiration)
	return dbValue, nil
}
