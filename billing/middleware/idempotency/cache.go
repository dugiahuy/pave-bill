package idempotency

import (
	"time"

	"encore.app/billing/model"
	"encore.dev/storage/cache"
)

// IdempotencyCluster is the cache cluster for idempotency
var IdempotencyCluster = cache.NewCluster("idempotency-cluster", cache.ClusterConfig{
	EvictionPolicy: cache.AllKeysLRU,
})

// IdempotencyCache is the keyspace for storing idempotency data
var IdempotencyCache = cache.NewStructKeyspace[model.IdempotencyKey, model.IdempotencyCacheEntry](
	IdempotencyCluster,
	cache.KeyspaceConfig{
		KeyPattern:    "idempotency/:Resource/:Key",
		DefaultExpiry: cache.ExpireIn(24 * time.Hour), // 24 hour expiry
	},
)
