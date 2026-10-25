package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"ylink-backend/internal/config"
	"ylink-backend/internal/middleware"
	"ylink-backend/internal/model"
	"ylink-backend/internal/pkg/logger"
	"ylink-backend/internal/pkg/mailer"
	redispkg "ylink-backend/internal/pkg/redis"
	"ylink-backend/internal/repo"
)

// Cron 调度深 Module：时机（分布式锁 + 跳过语义）/ 策略（窗口数字唯一 owner）/
// 投递（mail + TG 双扇出，非阻塞主流程）三层收拢。
//
// 调用方（worker 装配的 7 个公有定时入口）只表达调度意图：WithLock 包一层、
// 策略问 policy、提醒走 notifier。settings-JSON 解析、fallback 默认值、
// Redis 去重键形状、mail+TG 扇出失败语义一律不外泄。
//
// Depth 所在：lock 语义 + 窗口分类 + 双通道投递的可重试/可观测性。

// ---- 时机：分布式锁 ----

const cronLockTTL = 15 * time.Minute

// cronUnlockLua 属主校验释放锁：仅当锁值仍为本实例的 token 时 DEL。
// 任务执行超过锁 TTL 后锁已被继任实例重新持有，无属主校验的 DEL 会误删
// 继任者的锁，导致下一轮双实例并发执行同一任务。
var cronUnlockLua = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`)

// cronGuard 锁 seam：同一 job 单实例执行，占不到即跳过并打点。
type cronGuard struct {
	rdb *redis.Client
}

func (g *cronGuard) run(name string, fn func()) func() {
	return func() {
		ctx := context.Background()
		key := redispkg.Key("cron", "lock", name)
		token := uuid.NewString()
		ok, err := g.rdb.SetNX(ctx, key, token, cronLockTTL).Result()
		if err != nil {
			// Redis 故障与锁被他人持有是两种语义：故障必须 Error 级可观测，
			// 否则基础设施故障被静默降级成「本轮跳过」。
			logger.L().Error("cron lock acquire failed", zapS("job", name), zapE(err))
			middleware.CronJobInc(name, "skipped")
			return
		}
		if !ok {
			logger.L().Info("cron lock held by peer, skip", zapS("job", name))
			middleware.CronJobInc(name, "skipped")
			return
		}
		defer g.release(ctx, key, token)
		logger.L().Info("cron job start", zapS("job", name))
		start := time.Now()
		func() {
			defer func() {
				if r := recover(); r != nil {
					middleware.CronJobInc(name, "error")
					middleware.CronJobDuration(name, time.Since(start))
					panic(r)
				}
			}()
			fn()
			middleware.CronJobInc(name, "success")
			middleware.CronJobDuration(name, time.Since(start))
		}()
		logger.L().Info("cron job done", zapS("job", name))
	}
}

// release 属主校验释放（Lua 单原子）；失败只记日志，等锁 TTL 自然过期兜底。
func (g *cronGuard) release(ctx context.Context, key, token string) {
	if err := cronUnlockLua.Run(ctx, g.rdb, []string{key}, token).Err(); err != nil {
		logger.L().Error("cron lock release failed", zapS("job", key), zapE(err))
	}
}

// ---- 策略：窗口数字唯一 owner ----

// cronPolicy 策略 seam：settings-JSON 只在此解析，fallback 默认值只在此声明。
type cronPolicy struct {
	db    *gorm.DB
	repos *repo.Repos
}

// orderExpireMinutes 关单窗口（分）：order.expire_minutes，缺失/非法回退 30。
func (p *cronPolicy) orderExpireMinutes() int {
	minutes := 30
	type orderCfg struct {
		ExpireMinutes int `json:"expire_minutes"`
	}
	if raw, err := p.repos.Setting.Get(p.db, "order"); err == nil {
		var c orderCfg
		if json.Unmarshal([]byte(raw), &c) == nil && c.ExpireMinutes > 0 {
			minutes = c.ExpireMinutes
		}
	}
	return minutes
}

// commissionConfirmDays 佣金确认窗口（天）：invite.commission_confirm_days，缺失回退 3。
func (p *cronPolicy) commissionConfirmDays() int {
	days := 3
	type inviteCfg struct {
		CommissionConfirmDays int `json:"commission_confirm_days"`
	}
	if raw, err := p.repos.Setting.Get(p.db, "invite"); err == nil {
		var c inviteCfg
		if json.Unmarshal([]byte(raw), &c) == nil && c.CommissionConfirmDays > 0 {
			days = c.CommissionConfirmDays
		}
	}
	return days
}

// expireWindow 到期提醒窗口分类：剩余 (48h,72h]→"3d"，(0,24h]→"1d"，其余→""（不提醒）。
func expireWindow(remaining time.Duration) string {
	switch {
	case remaining > 48*time.Hour && remaining <= 72*time.Hour:
		return "3d"
	case remaining > 0 && remaining <= 24*time.Hour:
		return "1d"
	default:
		return ""
	}
}

// ---- 投递：mail + TG 双扇出 ----

// cronNotifier 投递 seam：模板渲染 + 邮件主通道 + TG 追加推送。
// 承诺：TG 失败仅记日志不阻断邮件主流程；异步投递带 recover，不泄漏 goroutine 恐慌。
type cronNotifier struct {
	db     *gorm.DB
	cfg    *config.Config
	mailer *mailer.Mailer
	tg     *TelegramService
}

// deliverExpire 同步投递一条到期提醒（渲染→邮件→TG 追加）。
func (n *cronNotifier) deliverExpire(u model.User) {
	if u.ExpiredAt == nil {
		return
	}
	vars := map[string]string{"expire_date": u.ExpiredAt.Format("2006-01-02")}
	subject, body, err := renderMailTemplate(n.db, n.cfg.App.Name, "expire_remind", vars)
	if err != nil {
		logger.L().Error("render expire mail", zapE(err))
		return
	}
	if err := n.mailer.Send(u.Email, subject, body); err != nil {
		logger.L().Error("send expire mail failed", zapS("email", u.Email), zapE(err))
	}
	// F12：已绑定 Telegram 的用户同步推送（失败仅记日志，不影响邮件主流程）
	if n.tg != nil {
		n.tg.NotifyUser(context.Background(), u, "您的订阅将于 "+vars["expire_date"]+" 到期，请及时续费以免影响使用。")
	}
}

// deliverTraffic 同步投递一条流量提醒。
func (n *cronNotifier) deliverTraffic(ctx context.Context, u model.User, percent int64) {
	vars := map[string]string{"percent": fmt.Sprint(percent)}
	subject, body, err := renderMailTemplate(n.db, n.cfg.App.Name, "traffic_remind", vars)
	if err != nil {
		logger.L().Error("render traffic mail", zapE(err))
		return
	}
	if err := n.mailer.Send(u.Email, subject, body); err != nil {
		logger.L().Error("send traffic mail failed", zapS("email", u.Email), zapE(err))
	}
	if n.tg != nil {
		n.tg.NotifyUser(ctx, u, "您的流量已使用 "+vars["percent"]+"%，请注意剩余流量。")
	}
}

// deliverAsync 异步投递：goroutine 内 recover，恐慌只记日志不崩 worker。
func (n *cronNotifier) deliverAsync(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.L().Error("cron notify panic", zap.Any("panic", r))
			}
		}()
		fn()
	}()
}
