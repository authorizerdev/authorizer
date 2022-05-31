package redis

import (
	"context"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
)

// RedisClient is the interface for redis client & redis cluster client
type RedisClient interface {
	HMSet(ctx context.Context, key string, values ...interface{}) *redis.BoolCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd
	HMGet(ctx context.Context, key string, fields ...string) *redis.SliceCmd
	HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	HGet(ctx context.Context, key, field string) *redis.StringCmd
	HGetAll(ctx context.Context, key string) *redis.StringStringMapCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd
}

type provider struct {
	ctx   context.Context
	store RedisClient
}

// NewRedisProvider returns a new redis provider
func NewRedisProvider(redisURL string) (*provider, error) {
	redisURLHostPortsList := strings.Split(redisURL, ",")

	if len(redisURLHostPortsList) > 1 {
		opt, err := redis.ParseURL(redisURLHostPortsList[0])
		if err != nil {
			log.Debug("error parsing redis url: ", err)
			return nil, err
		}
		urls := []string{opt.Addr}
		urlList := redisURLHostPortsList[1:]
		urls = append(urls, urlList...)
		clusterOpt := &redis.ClusterOptions{Addrs: urls}

		rdb := redis.NewClusterClient(clusterOpt)
		ctx := context.Background()
		_, err = rdb.Ping(ctx).Result()
		if err != nil {
			log.Debug("error connecting to redis: ", err)
			return nil, err
		}

		return &provider{
			ctx:   ctx,
			store: rdb,
		}, nil
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Debug("error parsing redis url: ", err)
		return nil, err
	}

	rdb := redis.NewClient(opt)
	ctx := context.Background()
	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		log.Debug("error connecting to redis: ", err)
		return nil, err
	}

	return &provider{
		ctx:   ctx,
		store: rdb,
	}, nil
}
