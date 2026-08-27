// Package config 负责加载配置：configs/config.yaml + APP_ 前缀环境变量覆盖。
package config

import (
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
	Site     SiteConfig     `mapstructure:"site"`
	Security SecurityConfig `mapstructure:"security"`
}

type AppConfig struct {
	Name    string `mapstructure:"name"`
	Env     string `mapstructure:"env"`
	Addr    string `mapstructure:"addr"`
	BaseURL string `mapstructure:"base_url"`
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

type SiteConfig struct {
	SiteName                  string `mapstructure:"site_name"`
	InviteCommissionRate      int    `mapstructure:"invite_commission_rate"`
	AgentCommissionRate       int    `mapstructure:"agent_commission_rate"`
	AgentRequiredValidInvites int    `mapstructure:"agent_required_valid_invites"`
	CommissionConfirmDays     int    `mapstructure:"commission_confirm_days"`
	OrderExpireMinutes        int    `mapstructure:"order_expire_minutes"`
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
	return &c, nil
}
