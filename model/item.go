package model

import (
	"time"
)

func NewItem(key, value string, ttl time.Time) *Item {
	return &Item{key: key, value: value, expiresAt: ttl}
}

// Item satisfies cache.Item interface
type Item struct {
	key       string
	value     string
	expiresAt time.Time
}

// IsExpired implements cache.Item.
func (i *Item) IsExpired() bool {
	if i.expiresAt.IsZero() {
		return false
	}

	return i.expiresAt.Before(time.Now().UTC())
}

// Key implements cache.Item.
func (i *Item) Key() string { return i.key }

// TTL implements cache.Item.
func (i *Item) TTL() time.Time { return i.expiresAt }

// Value implements cache.Item.
func (i *Item) Value() string { return i.value }
