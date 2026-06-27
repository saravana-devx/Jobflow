package cron

import (
	"context"
	"log"
	"time"

	redisx "jobflow/internal/redis"
)

// runIfLeader makes a cron tick run on only one replica. we grab a redis key
// with SET NX + ttl; whoever gets it is the leader for this tick, the rest skip.
// no manual unlock, the key just expires so another replica takes over if the
// leader dies. keep ttl close to the tick interval.
func runIfLeader(ctx context.Context, rdb *redisx.Redis, key string, ttl time.Duration, fn func()) {
	ok, err := rdb.Client.SetNX(ctx, key, "leader", ttl).Result()
	if err != nil {
		// redis down: skip this tick instead of letting every replica run
		log.Printf("cron: lease %s failed, skipping tick: %v", key, err)
		return
	}
	if !ok {
		// another replica holds it
		return
	}
	fn()
}
