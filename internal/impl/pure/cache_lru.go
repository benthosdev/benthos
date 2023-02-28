package pure

import (
	"context"
	"fmt"
	"sync"
	"time"

	lruv2 "github.com/hashicorp/golang-lru/v2"

	"github.com/benthosdev/benthos/v4/public/service"
)

const (
	lruCacheFieldCapLabel        = "cap"
	lruCacheFieldCapDefaultValue = 1000
	lruCacheFieldInitValuesLabel = "init_values"

	// specific to algorithm
	lruCacheFieldAlgorithmLabel         = "algorithm"
	lruCacheFieldAlgorithmValueStandard = "standard"
	lruCacheFieldAlgorithmValueARC      = "arc"
	lruCacheFieldAlgorithmValue2Q       = "two_queues"
	lruCacheFieldAlgorithmDefaultValue  = lruCacheFieldAlgorithmValueStandard

	// specific to algorithm two queues
	lruCacheField2QRecentRatioLabel        = "two_queues_recent_ratio"
	lruCacheField2QGhostRatioLabel         = "two_queues_ghost_ratio"
	lruCacheField2QRecentRatioDefaultValue = lruv2.Default2QRecentRatio
	lruCacheField2QGhostRatioDefaultValue  = lruv2.Default2QGhostEntries

	// optimistic
	lruCacheFieldOptimisticLabel        = "optimistic"
	lruCacheFieldOptimisticDefaultValue = false
)

func lruCacheConfig() *service.ConfigSpec {
	spec := service.NewConfigSpec().
		Stable().
		Summary(`Stores key/value pairs in a lru in-memory cache. This cache is therefore reset every time the service restarts.`).
		Description(`This provides the lru package which implements a fixed-size thread safe LRU cache.

It uses the package ` + "[`lru`](https://github.com/hashicorp/golang-lru/v2)" + ` 
		
The field ` + lruCacheFieldInitValuesLabel + ` can be used to prepopulate the memory cache with any number of key/value pairs:

` + "```yaml" + `
cache_resources:
  - label: foocache
    lru:
      cap: 1024
      init_values:
        foo: bar
` + "```" + `

These values can be overridden during execution, at which point the configured TTL is respected as usual.`).
		Field(service.NewIntField(lruCacheFieldCapLabel).
			Description("...").
			Default(lruCacheFieldCapDefaultValue)).
		Field(service.NewStringMapField(lruCacheFieldInitValuesLabel).
			Description("A table of key/value pairs that should be present in the cache on initialization. This can be used to create static lookup tables.").
			Default(map[string]string{}).
			Example(map[string]string{
				"Nickelback":       "1995",
				"Spice Girls":      "1994",
				"The Human League": "1977",
			})).
		Field(service.NewStringAnnotatedEnumField(lruCacheFieldAlgorithmLabel, map[string]string{
			lruCacheFieldAlgorithmValueStandard: "is a simple LRU cache. It is based on the LRU implementation in groupcache",
			lruCacheFieldAlgorithmValueARC:      "is an adaptive replacement cache. It tracks recent evictions as well as recent usage in both the frequent and recent caches. Its computational overhead is comparable to " + lruCacheFieldAlgorithmValue2Q + ", but the memory overhead is linear with the size of the cache. ARC has been patented by IBM.",
			lruCacheFieldAlgorithmValue2Q:       "tracks frequently used and recently used entries separately. This avoids a burst of accesses from taking out frequently used entries, at the cost of about 2x computational overhead and some extra bookkeeping.",
		}).
			Description("the lru cache implementation").
			Default(lruCacheFieldAlgorithmDefaultValue).
			Advanced()).
		Field(service.NewFloatField("two_queues_recent_ratio").
			Description("is the ratio of the " + lruCacheFieldAlgorithmValue2Q + " cache dedicated to recently added entries that have only been accessed once.").
			Default(lruCacheField2QRecentRatioDefaultValue).
			Advanced().
			Optional()).
		Field(service.NewFloatField("two_queues_ghost_ratio").
			Description("is the default ratio of ghost entries kept to track entries recently evicted on " + lruCacheFieldAlgorithmValue2Q + " cache.").
			Default(lruv2.Default2QGhostEntries).
			Advanced().
			Optional()).
		Field(service.NewBoolField(lruCacheFieldOptimisticLabel).
			Description("If true, we do not lock on read/write events. The lru package is thread-safe, however the ADD operation is not atomic.").
			Default(lruCacheFieldOptimisticDefaultValue).
			Advanced())

	return spec
}

func init() {
	err := service.RegisterCache(
		"lru", lruCacheConfig(),
		func(conf *service.ParsedConfig, mgr *service.Resources) (service.Cache, error) {
			f, err := lruMemCacheFromConfig(conf)
			if err != nil {
				return nil, err
			}
			return f, nil
		})
	if err != nil {
		panic(err)
	}
}

func lruMemCacheFromConfig(conf *service.ParsedConfig) (*lruCacheAdapter, error) {
	capacity, err := conf.FieldInt(lruCacheFieldCapLabel)
	if err != nil {
		return nil, err
	}

	initValues, err := conf.FieldStringMap(lruCacheFieldInitValuesLabel)
	if err != nil {
		return nil, err
	}

	algorithm, err := conf.FieldString(lruCacheFieldAlgorithmLabel)
	if err != nil {
		return nil, err
	}

	var recentRatioPtr, ghostRatioPtr *float64

	if conf.Contains(lruCacheField2QRecentRatioLabel) || conf.Contains(lruCacheField2QGhostRatioLabel) {
		recentRatio, err := conf.FieldFloat(lruCacheField2QRecentRatioLabel)
		if err != nil {
			return nil, err
		}

		ghostRatio, err := conf.FieldFloat(lruCacheField2QGhostRatioLabel)
		if err != nil {
			return nil, err
		}

		recentRatioPtr = &recentRatio
		ghostRatioPtr = &ghostRatio
	}

	optimistic, err := conf.FieldBool(lruCacheFieldOptimisticLabel)
	if err != nil {
		return nil, err
	}

	return lruMemCache(capacity, algorithm, initValues, recentRatioPtr, ghostRatioPtr, optimistic)
}

//------------------------------------------------------------------------------

var (
	errInvalidLRUCacheCapacityValue = fmt.Errorf("invalid lru cache parameter capacity: must be bigger than 0")
)

func lruMemCache(capacity int,
	algorithm string,
	initValues map[string]string,
	recentRatio, ghostRatio *float64,
	optimistic bool) (ca *lruCacheAdapter, err error) {
	if capacity <= 0 {
		return nil, errInvalidLRUCacheCapacityValue
	}

	var inner lruCache

	switch algorithm {
	case lruCacheFieldAlgorithmValueStandard:
		var c *lruv2.Cache[string, []byte]
		c, err = lruv2.New[string, []byte](capacity)
		if err != nil {
			return
		}

		inner = &lruv2SimpleCacheAdaptor{
			Cache: c,
		}

	case lruCacheFieldAlgorithmValueARC:
		inner, err = lruv2.NewARC[string, []byte](capacity)
		if err != nil {
			return
		}

	case lruCacheFieldAlgorithmValue2Q:
		if recentRatio != nil && ghostRatio != nil {
			inner, err = lruv2.New2QParams[string, []byte](capacity, *recentRatio, *ghostRatio)
		} else {
			inner, err = lruv2.New2Q[string, []byte](capacity)
		}

		if err != nil {
			return
		}
	default:
		return nil, fmt.Errorf("algorithm %q not supported. the supported values are %q, %q and %q", algorithm,
			lruCacheFieldAlgorithmValueStandard, lruCacheFieldAlgorithmValueARC, lruCacheFieldAlgorithmValue2Q)
	}

	for k, v := range initValues {
		inner.Add(k, []byte(v))
	}

	ca = &lruCacheAdapter{
		inner:      inner,
		optimistic: optimistic,
	}

	return ca, nil
}

//------------------------------------------------------------------------------

var (
	_ lruCache = (*lruv2SimpleCacheAdaptor)(nil)
	_ lruCache = (*lruv2.TwoQueueCache[string, []byte])(nil)
	_ lruCache = (*lruv2.ARCCache[string, []byte])(nil)
)

type lruCache interface {
	Peek(key string) (value []byte, ok bool)
	Get(key string) (value []byte, ok bool)
	Add(key string, value []byte)
	Remove(key string)
}

type lruv2SimpleCacheAdaptor struct {
	*lruv2.Cache[string, []byte]
}

func (ad *lruv2SimpleCacheAdaptor) Add(key string, value []byte) {
	_ = ad.Cache.Add(key, value)
}

func (ad *lruv2SimpleCacheAdaptor) Remove(key string) {
	_ = ad.Cache.Remove(key)
}

//------------------------------------------------------------------------------

var _ service.Cache = (*lruCacheAdapter)(nil)

type lruCacheAdapter struct {
	inner lruCache

	optimistic bool

	sync.RWMutex
}

func (ca *lruCacheAdapter) Get(_ context.Context, key string) ([]byte, error) {
	unlock := func() {}
	if !ca.optimistic {
		ca.RWMutex.RLock()

		unlock = ca.RWMutex.RUnlock
	}

	value, ok := ca.inner.Get(key)

	unlock()

	if !ok {
		return nil, service.ErrKeyNotFound
	}

	return value, nil
}

func (ca *lruCacheAdapter) Set(_ context.Context, key string, value []byte, _ *time.Duration) error {
	unlock := func() {}
	if !ca.optimistic {
		ca.RWMutex.Lock()

		unlock = ca.RWMutex.Unlock
	}

	ca.inner.Add(key, value)

	unlock()

	return nil
}

func (ca *lruCacheAdapter) Add(ctx context.Context, key string, value []byte, ttl *time.Duration) error {
	unlock := func() {}
	if !ca.optimistic {
		ca.RWMutex.Lock()

		unlock = ca.RWMutex.Unlock
	}

	_, ok := ca.inner.Peek(key)
	if ok {
		unlock()

		return service.ErrKeyAlreadyExists
	}

	ca.inner.Add(key, value)

	unlock()

	return nil
}

func (ca *lruCacheAdapter) Delete(_ context.Context, key string) error {
	unlock := func() {}
	if !ca.optimistic {
		ca.RWMutex.Lock()

		unlock = ca.RWMutex.Unlock
	}

	ca.inner.Remove(key)

	unlock()

	return nil
}

func (ca *lruCacheAdapter) Close(_ context.Context) error {
	return nil
}
