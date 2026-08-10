// Package service 实现业务逻辑、事务边界与领域规则（单测重点）。
package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	redispkg "ylink/internal/pkg/redis"
	"ylink/internal/repo"
)

const settingsCacheTTL = 60 * time.Second

// SettingService 站点配置读取（Redis 缓存 60s，管理端变更即失效）。
type SettingService struct {
	db    *gorm.DB
	rdb   *redis.Client
	repos *repo.Repos
}

func NewSettingService(db *gorm.DB, rdb *redis.Client, repos *repo.Repos) *SettingService {
	return &SettingService{db: db, rdb: rdb, repos: repos}
}

// GetString 读取配置项原始 JSON 字符串。
func (s *SettingService) GetString(ctx context.Context, key string) (string, error) {
	cacheKey := redispkg.Key("cache", "settings", key)
	if v, err := s.rdb.Get(ctx, cacheKey).Result(); err == nil {
		return v, nil
	}
	v, err := s.repos.Setting.Get(s.db, key)
	if err != nil {
		return "", err
	}
	s.rdb.Set(ctx, cacheKey, v, settingsCacheTTL)
	return v, nil
}

// GetJSON 读取并反序列化配置项。
func (s *SettingService) GetJSON(ctx context.Context, key string, out any) error {
	raw, err := s.GetString(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), out)
}

// Invalidate 配置变更后使缓存失效。
func (s *SettingService) Invalidate(ctx context.Context, key string) {
	s.rdb.Del(ctx, redispkg.Key("cache", "settings", key))
}
