package redis

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
)

const (
	dialTimeout = 60 * time.Second
)

// Dependencies struct for redis provider
type Dependencies struct {
	Log *zerolog.Logger
}

// RedisClient is the interface for redis client & redis cluster client
type RedisClient interface {
	HMSet(ctx context.Context, key string, values ...interface{}) *redis.BoolCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd
	HMGet(ctx context.Context, key string, fields ...string) *redis.SliceCmd
	HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	HGet(ctx context.Context, key, field string) *redis.StringCmd
	HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd
	Keys(ctx context.Context, pattern string) *redis.StringSliceCmd
}

type provider struct {
	config       config.Config
	dependencies Dependencies

	ctx   context.Context
	store RedisClient
}

// NewRedisProvider returns a new redis provider
func NewRedisProvider(cfg config.Config, deps Dependencies) (*provider, error) {
	redisURLHostPortsList := strings.Split(cfg.RedisURL, ",")
	if len(redisURLHostPortsList) > 1 {
		opt, err := redis.ParseURL(redisURLHostPortsList[0])
		if err != nil {
			deps.Log.Debug().Err(err).Msg("error parsing redis url")
			return nil, err
		}
		urls := []string{opt.Addr}
		urlList := redisURLHostPortsList[1:]
		urls = append(urls, urlList...)
		clusterOpt := &redis.ClusterOptions{Addrs: urls, DialTimeout: dialTimeout}
		rdb := redis.NewClusterClient(clusterOpt)
		ctx := context.Background()
		_, err = rdb.Ping(ctx).Result()
		if err != nil {
			deps.Log.Debug().Err(err).Msg("error connecting to redis")
			return nil, err
		}

		return &provider{
			config:       cfg,
			dependencies: deps,
			ctx:          ctx,
			store:        rdb,
		}, nil
	}

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		deps.Log.Debug().Err(err).Msg("error parsing redis url")
		return nil, err
	}
	opt.DialTimeout = dialTimeout
	rdb := redis.NewClient(opt)
	ctx := context.Background()
	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		deps.Log.Debug().Err(err).Msg("error connecting to redis")
		return nil, err
	}
	return &provider{
		ctx:   ctx,
		store: rdb,
	}, nil
}
