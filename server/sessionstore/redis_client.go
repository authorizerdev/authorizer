package sessionstore

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisSessionClient interface {
	HMSet(ctx context.Context, key string, values ...interface{}) *redis.BoolCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd
	HMGet(ctx context.Context, key string, fields ...string) *redis.SliceCmd
	HGetAll(ctx context.Context, key string) *redis.StringStringMapCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd
}
