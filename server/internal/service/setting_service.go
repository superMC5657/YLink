// Package service 实现业务逻辑、事务边界与领域规则（单测重点）。
package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"ylink-backend/internal/model"
	redispkg "ylink-backend/internal/pkg/redis"
	"ylink-backend/internal/repo"
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

// jsonIntOf settings-JSON 整数项解析唯一口：字段缺失/类型不符/解析失败/非正值
// 一律回退默认值（收敛前 5 处各自「读 raw + Unmarshal + 回退」的两式访问）。
func jsonIntOf(raw, field string, def int) int {
	var m map[string]json.Number
	if json.Unmarshal([]byte(raw), &m) != nil {
		return def
	}
	v, err := m[field].Int64()
	if err != nil || v <= 0 {
		return def
	}
	return int(v)
}

// GetInt settings-JSON 整数项读取唯一口（Redis 60s 缓存 + 默认值回退）。
// 读取失败（DB/Redis 异常）同样回退默认值，不阻断调用方主流程。
func (s *SettingService) GetInt(ctx context.Context, key, field string, def int) int {
	raw, err := s.GetString(ctx, key)
	if err != nil {
		return def
	}
	return jsonIntOf(raw, field, def)
}

// commissionRateFor 取佣金比例（代理商取 agent 比例；invite 配置缺失时回退默认值）。
// order 下单与 invite 总览共用，避免两处重复实现。
func commissionRateFor(ctx context.Context, set *SettingService, inviterRole int) int {
	if inviterRole == model.RoleAgent {
		return set.GetInt(ctx, "invite", "agent_commission_rate", 50)
	}
	return set.GetInt(ctx, "invite", "commission_rate", 40)
}
