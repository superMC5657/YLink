// Package redis 提供 go-redis 客户端初始化。
package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"nanocloud/internal/config"
)

// New 创建 Redis 客户端并探测连通性。
func New(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return rdb, nil
}

// Key 统一构造带命名空间的 Key。
func Key(parts ...string) string {
	const sep = ":"
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}

// SetString 便捷写入带过期时间的字符串。
func SetString(ctx context.Context, rdb *redis.Client, key, value string, ttl time.Duration) error {
	return rdb.Set(ctx, key, value, ttl).Err()
}
