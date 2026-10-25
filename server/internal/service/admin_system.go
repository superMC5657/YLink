package service

import (
	"context"

	"ylink-backend/internal/model"
)

// 配置域 module：站点配置读写。

// ListSettings GET /admin/settings。
// 直连豁免：settings 键值整表读（无单键谓词，repo 仅有按 key Get/Set 形状）。
func (m *adminSystemModule) ListSettings(ctx context.Context) ([]model.AdminSettingsResp, error) {
	var rows []model.Setting
	if err := m.db.Order("key ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]model.AdminSettingsResp, 0, len(rows))
	for _, r := range rows {
		out = append(out, model.AdminSettingsResp{Key: r.Key, Value: r.Value})
	}
	return out, nil
}

// SaveSetting PUT /admin/settings（写后失效 Redis 缓存）。
func (m *adminSystemModule) SaveSetting(ctx context.Context, key, value string) error {
	if err := m.repos.Setting.Set(m.db, key, value); err != nil {
		return err
	}
	m.set.Invalidate(ctx, key)
	return nil
}
