package inmem

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/kylycht/kviku/cache"
	"github.com/sirupsen/logrus"
)

const (
	defaultCleanupInterval time.Duration = time.Second * 10
)

func New(stopC chan struct{}) cache.Cache {
	cc := &inMemoryCache{
		lock:    sync.RWMutex{},
		storage: map[string]cache.Item{},
		timerC:  time.NewTicker(defaultCleanupInterval),
		stopC:   stopC,
	}

	// start cleaner
	go cc.janitor()

	return cc
}

type inMemoryCache struct {
	lock    sync.RWMutex          // rw lock guarding storage map
	storage map[string]cache.Item // underlying storage of the cache
	timerC  *time.Ticker          // time ticker that is responsible to trigger clean up
	stopC   chan struct{}         // channel to stop cleaning processor
}

// Get implements cache.Cache.
func (i *inMemoryCache) Get(key string) (string, bool) {
	i.lock.RLock()
	defer i.lock.RUnlock()

	v := i.load(key)
	if v == nil {
		return "", false
	}

	return v.Value(), true
}

// Save implements cache.Cache.
// If key is already present, it will overwrite exisiting value
func (i *inMemoryCache) Save(item cache.Item) error {
	i.lock.Lock()
	i.store(item)
	i.lock.Unlock()

	return nil
}

func (i *inMemoryCache) load(key string) cache.Item {
	logrus.Debug("loading item from cache")
	v, isPresent := i.storage[key]
	if !isPresent {
		return nil
	}

	if !v.IsExpired() {
		logrus.Debug("found unexpired item")
		return v
	}

	return nil
}

func (i *inMemoryCache) store(item cache.Item) {
	logrus.Debug("storing item into cache")
	i.storage[item.Key()] = item
}

func (i *inMemoryCache) evict() int {
	i.lock.Lock()
	defer i.lock.Unlock()

	var evicted atomic.Int32

	for k, v := range i.storage {
		if v.IsExpired() {
			evicted.Add(1)
			delete(i.storage, k)
		}
	}
	return int(evicted.Load())
}

func (i *inMemoryCache) janitor() {
	for {
		select {
		case <-i.timerC.C:
			logrus.Debug("starting eviction")
			n := i.evict()
			logrus.Debug("evicted: ", n)
		case <-i.stopC:
			i.timerC.Stop()
		}
	}
}
