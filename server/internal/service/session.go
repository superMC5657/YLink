package service

import (
	"context"

	"github.com/redis/go-redis/v9"

	redispkg "ylink-backend/internal/pkg/redis"
)

// session.go 现为会话吊销深 Module 的薄 adapter：历史小写 helper 保留签名，
// 一律转发到 SessionRevoker / redispkg 的拥有 seam，新代码请直接表达吊销意图
// （RevokeReason*），不得再手拼 Key。

// sessionVersion 读取用户当前会话版本号；Key 不存在视为 0。
func sessionVersion(ctx context.Context, rdb *redis.Client, userID int64) int64 {
	return NewSessionRevoker(rdb, 0).CurrentVersion(ctx, userID)
}

// bumpSessionVersion 使该用户全部已签发 access token 立即失效（401）。
// 触发点：封禁/解封、角色变更、代理审批通过、代理商降级、找回密码、登出。
func bumpSessionVersion(ctx context.Context, rdb *redis.Client, userID int64) {
	_ = NewSessionRevoker(rdb, 0).RevokeAll(ctx, userID, RevokeReasonBan)
}

// refreshKey 历史 helper：转发到深 Module 拥有的 RefreshKey。
func refreshKey(userID int64, jti string) string {
	return redispkg.RefreshKey(userID, jti)
}
