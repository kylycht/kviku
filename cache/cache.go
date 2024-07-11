package cache

import "time"

const (
	DefaultTTL time.Duration = 0
)

// Item interface describes methods
// that needs to be satisfied
// for objects stored in Cache
type Item interface {
	// Key of the storing value
	Key() string
	// Value stored
	Value() string
	// IsExpired indicates if Item
	// is expired
	IsExpired() bool
	// TTL when item will be expired
	TTL() time.Time
}

// Cache interface desribes methods specifications
// required for caching objects
type Cache interface {
	// Get retrives value V from cache
	// indexed by key K
	//
	// Boolean return value indicates if
	// any match is present for key K
	Get(key string) (string, bool)

	// Save stores passed value V with
	// mapping key K with time to live
	// specified with ttl argument
	Save(Item) error
}
