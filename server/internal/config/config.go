// Package config 负责加载配置：configs/config.yaml + APP_ 前缀环境变量覆盖。
package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 为全部运行配置。
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	SMTP     SMTPConfig     `mapstructure:"smtp"`
	Payment  PaymentConfig  `mapstructure:"payment"`
	CORS     CORSConfig     `mapstructure:"cors"`
	Log      LogConfig      `mapstructure:"log"`
	Security SecurityConfig `mapstructure:"security"`
	Update   UpdateConfig   `mapstructure:"update"`
}

type AppConfig struct {
	Name    string `mapstructure:"name"`
	Env     string `mapstructure:"env"`
	Addr    string `mapstructure:"addr"`
	BaseURL string `mapstructure:"base_url"`
	Version string `mapstructure:"version"` // 后端版本号（F20 版本检查展示；构建/部署注入）
}

func (a AppConfig) IsProduction() bool { return a.Env == "production" }

type DatabaseConfig struct {
	DSN     string `mapstructure:"dsn"`
	MaxOpen int    `mapstructure:"max_open"`
	MaxIdle int    `mapstructure:"max_idle"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type JWTConfig struct {
	Secret     string        `mapstructure:"secret"`
	AccessTTL  time.Duration `mapstructure:"access_ttl"`
	RefreshTTL time.Duration `mapstructure:"refresh_ttl"`
}

type SMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	FromName string `mapstructure:"from_name"`
}

type PaymentConfig struct {
	Epay EpayConfig `mapstructure:"epay"`
}

type EpayConfig struct {
	Gateway string   `mapstructure:"gateway"`
	PID     string   `mapstructure:"pid"`
	Key     string   `mapstructure:"key"`
	Methods []string `mapstructure:"methods"`
}

type CORSConfig struct {
	AllowOrigins []string `mapstructure:"allow_origins"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
	Dir   string `mapstructure:"dir"`
}

// SecurityConfig 安全部署项（F22）：后台路径定制 / safe_mode 域名白名单 / 订阅路径定制。
// 全部启动注入（config.yaml + APP_SECURITY_* 环境变量），不落库，避免改路径后需迁移数据库。
type SecurityConfig struct {
	AdminPath     string   `mapstructure:"admin_path"`     // 管理端 API 路径段，默认 admin
	SubscribePath string   `mapstructure:"subscribe_path"` // 订阅下发路径段，默认 client
	SafeMode      bool     `mapstructure:"safe_mode"`      // 开启后非白名单域名的请求一律 403
	SafeDomains   []string `mapstructure:"safe_domains"`   // 额外白名单域名（App.BaseURL 的 host 自动纳入）
}

// UpdateConfig 在线更新（F20 子集）：版本检查走可选的远端 manifest JSON，
// 自动执行升级不立项（spec F20）。manifest 响应格式：{"version":"x.y.z","notes":"..."}。
type UpdateConfig struct {
	ManifestURL string `mapstructure:"manifest_url"` // 更新源地址（空=不做远端检查）
}

// AdminPathOrDefault 返回管理端路径段（空值兜底 admin）。
func (s SecurityConfig) AdminPathOrDefault() string {
	if s.AdminPath == "" {
		return "admin"
	}
	return s.AdminPath
}

// SubscribePathOrDefault 返回订阅路径段（空值兜底 client）。
func (s SecurityConfig) SubscribePathOrDefault() string {
	if s.SubscribePath == "" {
		return "client"
	}
	return s.SubscribePath
}

// Load 读取配置文件；path 为空时使用默认 configs/config.yaml。
func Load(path string) (*Config, error) {
	v := viper.New()
	if path == "" {
		path = "configs/config.yaml"
	}
	v.SetConfigFile(path)
	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	// app.* 经 AutomaticEnv 映射为 APP_APP_*，而部署模板与 dev 脚本统一用 APP_ENV 短名；
	// 不显式绑定则生产 env 标识被静默忽略（env 一直是文件里的 development）。
	if err := v.BindEnv("app.env", "APP_ENV"); err != nil {
		return nil, err
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, err
	}
	if c.Database.MaxOpen == 0 {
		c.Database.MaxOpen = 50
	}
	if c.Database.MaxIdle == 0 {
		c.Database.MaxIdle = 10
	}
	if err := c.validateProd(); err != nil {
		return nil, err
	}
	return &c, nil
}

// 仓库内公开的开发默认密钥：生产模式命中任意一个即拒绝启动，
// 防止部署漏配环境变量时静默使用可伪造 token 的公开口令。
const (
	devJWTSecretPrefix  = "dev-only-secret-" // configs/config.yaml 旧 JWT 默认值
	devEnvTemplateValue = "please-change-me" // server/.env.example 模板占位
	devDBPassword       = "ylink_root"       // 本地编排 PostgreSQL 默认口令
	devRedisPassword    = "ylink_redis"      // 本地编排 Redis 默认口令
)

// validateProd 生产 fail-fast：JWT/DB/Redis 密钥为空、为 ${...} 占位符
// 或仍为已知开发默认值时拒绝启动；开发模式直接放行。
func (c *Config) validateProd() error {
	if !c.App.IsProduction() {
		return nil
	}
	checks := []struct {
		item string
		env  string
		val  string
	}{
		{"jwt.secret", "APP_JWT_SECRET", c.JWT.Secret},
		{"database.dsn 的 password", "APP_DATABASE_DSN", dsnPassword(c.Database.DSN)},
		{"redis.password", "APP_REDIS_PASSWORD", c.Redis.Password},
	}
	for _, ck := range checks {
		if isMissingOrDevSecret(ck.val) {
			return fmt.Errorf("生产模式拒绝启动：%s 为空、占位符或仍为开发默认值，请经环境变量 %s 注入真实密钥", ck.item, ck.env)
		}
	}
	return nil
}

// isMissingOrDevSecret 判断密钥缺失（空串或 ${...} 占位符）或命中已知开发默认值。
func isMissingOrDevSecret(v string) bool {
	if v == "" || (strings.HasPrefix(v, "${") && strings.HasSuffix(v, "}")) {
		return true
	}
	return strings.HasPrefix(v, devJWTSecretPrefix) ||
		strings.HasPrefix(v, devEnvTemplateValue) ||
		v == devDBPassword ||
		v == devRedisPassword
}

// dsnPassword 提取 DSN 中的 password：postgres:// URL 形式取 userinfo 口令，
// 其余按空格分词取 password= 值（含引号时去引号），提取不到返回空串。
func dsnPassword(dsn string) string {
	if u, err := url.Parse(dsn); err == nil && u.User != nil {
		if pw, ok := u.User.Password(); ok {
			return pw
		}
	}
	for _, tok := range strings.Fields(dsn) {
		if v, ok := strings.CutPrefix(tok, "password="); ok {
			return strings.Trim(v, `'"`)
		}
	}
	return ""
}
