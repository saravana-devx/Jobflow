package auth

import (
	"context"
	"time"

	"jobflow/internal/redis"
)

type JTIStore struct {
	rdb *redis.Redis
}

func NewJTIStore(rdb *redis.Redis) *JTIStore {
	return &JTIStore{rdb: rdb}
}

func jtiKey(jti string) string {
	return "jti:revoked:" + jti
}

// Revoke marks a jti as revoked until ttl elapses. ttl should be the time
// remaining until the access token's exp — no point keeping the entry longer
// than the token would be valid anyway. If ttl <= 0 the token is already
// expired and there is nothing to do.
func (s *JTIStore) Revoke(ctx context.Context, jti string, ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}
	return s.rdb.Client.Set(ctx, jtiKey(jti), 1, ttl).Err()
}

func (s *JTIStore) IsRevoked(ctx context.Context, jti string) (bool, error) {
	n, err := s.rdb.Client.Exists(ctx, jtiKey(jti)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
