package service

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/errs"
	redispkg "ylink-backend/internal/pkg/redis"
)

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// 会话吊销深 Module：白名单 refresh:{uid}:{jti} + 会话版本号SV auth:ver:{uid} +
// kill-Hash auth:kill:{uid} 三套语义的唯一拥有者。
//
// 调用方（admin 封禁/改角色、代理审批/降级、找回密码、登出、用户踢会话）只表达
// 吊销意图（RevocationReason），由本 Module 决定写哪些键、bump 哪个版本、
// 删哪些白名单条目。Key 名与 Hash 字段一律不外泄：外部一律经由本文件 seam，
// 不得手拼 "refresh:" / "auth:ver:" / "auth:kill:" 前缀。
//
// Depth 所在：一次撤销、处处失效（白名单删 + SV bump + kill 标记的组合语义），
// 以及 SCAN 列举 + 元数据解析 + 当前会话标记的读语义。

// RevocationReason 吊销意图：调用方声明发生了什么，Module 决定如何失效。
type RevocationReason string

const (
	// RevokeReasonBan 封禁/解封：全部会话立即失效。
	RevokeReasonBan RevocationReason = "ban"
	// RevokeReasonRoleChange 角色变更：旧 role 快照立即失效。
	RevokeReasonRoleChange RevocationReason = "role_change"
	// RevokeReasonAgentApprove 代理审批通过：同角色变更。
	RevokeReasonAgentApprove RevocationReason = "agent_approve"
	// RevokeReasonAgentDowngrade 代理降级/月度复核：同角色变更。
	RevokeReasonAgentDowngrade RevocationReason = "agent_downgrade"
	// RevokeReasonPasswordReset 找回密码/改密：全部会话失效。
	RevokeReasonPasswordReset RevocationReason = "password_reset"
	// RevokeReasonLogout 登出：当前 refresh + 全局 access 失效。
	RevokeReasonLogout RevocationReason = "logout"
)

// SessionRevoker 会话吊销深 Module 的窄 Interface（写 seam + 读 seam）。
// killTTL 为 kick 标记保留时长，0 即用 redispkg.KillTTL（随 refresh TTL）。
type SessionRevoker struct {
	rdb     *redis.Client
	killTTL time.Duration
}

// NewSessionRevoker 构造吊销器；rdb 为空时方法均为 no-op（保测试/降级路径）。
func NewSessionRevoker(rdb *redis.Client, killTTL time.Duration) *SessionRevoker {
	if killTTL <= 0 {
		killTTL = redispkg.KillTTL
	}
	return &SessionRevoker{rdb: rdb, killTTL: killTTL}
}

// RevokeAll 意图式全量吊销：删该用户全部 refresh 白名单 + bump 会话版本号。
// 适用于封禁/解封、角色变更、代理审批/降级、找回密码。
func (v *SessionRevoker) RevokeAll(ctx context.Context, userID int64, _ RevocationReason) error {
	if v == nil || v.rdb == nil {
		return nil
	}
	pattern := redispkg.Key("refresh", itoa(userID), "*")
	iter := v.rdb.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		v.rdb.Del(ctx, iter.Val())
	}
	// bump 会话版本号：已签发 access 立即失效；失败不阻塞主流程
	_ = v.rdb.Incr(ctx, redispkg.SessionVersionKey(userID)).Err()
	return iter.Err()
}

// RevokeOthers 吊销除 keepJTI 外的全部白名单（改密：本机会话保留）。
func (v *SessionRevoker) RevokeOthers(ctx context.Context, userID int64, keepJTI string) error {
	if v == nil || v.rdb == nil {
		return nil
	}
	keep := redispkg.RefreshKey(userID, keepJTI)
	pattern := redispkg.Key("refresh", itoa(userID), "*")
	iter := v.rdb.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		if iter.Val() != keep {
			v.rdb.Del(ctx, iter.Val())
		}
	}
	return iter.Err()
}

// LogoutCurrent 登出：删当前 refresh 白名单 + bump 会话版本号。
func (v *SessionRevoker) LogoutCurrent(ctx context.Context, userID int64, jti string) {
	if v == nil || v.rdb == nil {
		return
	}
	if jti != "" {
		v.rdb.Del(ctx, redispkg.RefreshKey(userID, jti))
	}
	_ = v.rdb.Incr(ctx, redispkg.SessionVersionKey(userID)).Err()
}

// RevokeOne 单会话踢下线：删 refresh 白名单 + 写 kill-Hash（access 立即失效，
// 其余会话不受影响）。当前会话不可自踢，由调用方先判 currentJTI。
func (v *SessionRevoker) RevokeOne(ctx context.Context, userID int64, jti, currentJTI string) error {
	if v == nil || v.rdb == nil {
		return errs.ErrNotFound
	}
	if jti == "" {
		return errs.ErrNotFound
	}
	if jti == currentJTI {
		return errs.New(40000, "不能踢除当前登录会话")
	}
	n, err := v.rdb.Del(ctx, redispkg.RefreshKey(userID, jti)).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		return errs.ErrNotFound
	}
	killKey := redispkg.SessionKillKey(userID)
	v.rdb.HSet(ctx, killKey, jti, 1)
	v.rdb.Expire(ctx, killKey, v.killTTL)
	return nil
}

// SessionView 读 seam 的答案形状（含空集语义，调用方不再分支 nil）。
type SessionView = model.UserSessionItem

// List 列举活跃会话（refresh 白名单维度 + 元数据 JSON 解析 + 当前标记）。
func (v *SessionRevoker) List(ctx context.Context, userID int64, currentJTI string) ([]SessionView, error) {
	out := make([]SessionView, 0, 4)
	if v == nil || v.rdb == nil {
		return out, nil
	}
	pattern := redispkg.Key("refresh", itoa(userID), "*")
	iter := v.rdb.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		jti := key[strings.LastIndex(key, ":")+1:]
		if jti == "" {
			continue
		}
		raw, err := v.rdb.Get(ctx, key).Result()
		if err != nil {
			continue
		}
		var meta refreshMeta
		if json.Unmarshal([]byte(raw), &meta) != nil {
			continue
		}
		item := model.UserSessionItem{JTI: jti, Current: jti == currentJTI}
		item.IP = meta.IP
		item.UserAgent = meta.UserAgent
		ts := meta.CreatedAt
		item.CreatedAt = &ts
		out = append(out, item)
	}
	return out, iter.Err()
}

// Allowed 读 seam：该 access 是否仍有效（SV 匹配且未被单踢）。
func (v *SessionRevoker) Allowed(ctx context.Context, userID int64, jti string, sv int64) bool {
	if v == nil || v.rdb == nil {
		return true
	}
	ok, _ := redispkg.SessionAccessAllowed(ctx, v.rdb, userID, jti, sv)
	return ok
}

// CurrentVersion 读 seam：用户当前会话版本号（Key 不存在视为 0）。
func (v *SessionRevoker) CurrentVersion(ctx context.Context, userID int64) int64 {
	if v == nil || v.rdb == nil {
		return 0
	}
	n, err := v.rdb.Get(ctx, redispkg.SessionVersionKey(userID)).Int64()
	if err != nil {
		return 0
	}
	return n
}
