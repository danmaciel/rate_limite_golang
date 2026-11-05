package strategy

import "time"

type PersistenceStrategy interface {
	Set(key string, value interface{}) error
	Get(key string, dest interface{}) error
	GetCount(key string) int
	Delete(key string) error
	// Allow atomically increments a fixed-window counter shared across processes.
	// The first request after expiration starts a new window.
	Allow(key string, limit int, window, block time.Duration) (bool, error)
}
