package auth

// import (
// 	"context"
// 	"time"

// 	"jobflow/internal/redis"
// )

// type TokenStore struct {
// 	rdb *redis.Redis
// }

// func NewTokenStore(rdb *redis.Redis) *TokenStore {
// 	return &TokenStore{rdb: rdb}
// }

// func (s *TokenStore) SetRefreshToken(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error {
// 	return s.rdb.Client.Set(ctx, tokenHash, userID, time.Until(expiresAt)).Err()
// }

// func (s *TokenStore) GetUserByRefreshToken(ctx context.Context, tokenHash string) (string, error) {
// 	return s.rdb.Client.Get(ctx, tokenHash).Result()
// }

// func (s *TokenStore) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
// 	return s.rdb.Client.Del(ctx, tokenHash).Err()
// }
