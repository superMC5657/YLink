package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 缝测试：生产 fail-fast（占位/开发默认密钥拒启）与 APP_ENV 短名绑定。
// 最怕的失败=生产部署漏配密钥静默启动（公开口令签发 JWT 可伪造任意用户）、
// APP_ENV 不生效导致 fail-fast 形同虚设；开发模式必须不受影响。

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

// neutralizeEnv 清空可能存在的同名环境变量（viper 默认忽略空值，文件值可透传）。
func neutralizeEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"APP_ENV", "APP_APP_ENV", "APP_JWT_SECRET", "APP_DATABASE_DSN", "APP_REDIS_PASSWORD"} {
		t.Setenv(k, "")
	}
}

func prodSecretEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_JWT_SECRET", "real-jwt-secret-0123456789abcdef0123")
	t.Setenv("APP_DATABASE_DSN", "host=db port=5432 user=ylink password=real-pass dbname=ylink sslmode=disable")
	t.Setenv("APP_REDIS_PASSWORD", "real-redis-pass")
}

func prodBase() *Config {
	c := &Config{}
	c.App.Env = "production"
	c.JWT.Secret = "real-jwt-secret-0123456789abcdef0123"
	c.Database.DSN = "host=db port=5432 user=ylink password=real-pass dbname=ylink sslmode=disable"
	c.Redis.Password = "real-redis-pass"
	return c
}

func TestValidateProdRejectsMissingOrDevSecrets(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Config)
		envKey string
	}{
		{"jwt 为空", func(c *Config) { c.JWT.Secret = "" }, "APP_JWT_SECRET"},
		{"jwt 为占位符", func(c *Config) { c.JWT.Secret = "${APP_JWT_SECRET}" }, "APP_JWT_SECRET"},
		{"jwt 为 config.yaml 旧开发默认", func(c *Config) { c.JWT.Secret = "dev-only-secret-please-override-0123456789abcdef" }, "APP_JWT_SECRET"},
		{"jwt 为 env 模板占位", func(c *Config) { c.JWT.Secret = "please-change-me-32bytes-secret-key!!" }, "APP_JWT_SECRET"},
		{"db 口令缺失", func(c *Config) { c.Database.DSN = "${APP_DATABASE_DSN}" }, "APP_DATABASE_DSN"},
		{"db 口令为开发默认", func(c *Config) {
			c.Database.DSN = "host=db port=5432 user=ylink password=ylink_root dbname=ylink sslmode=disable"
		}, "APP_DATABASE_DSN"},
		{"db URL 形式口令为开发默认", func(c *Config) { c.Database.DSN = "postgres://ylink:ylink_root@db:5432/ylink?sslmode=disable" }, "APP_DATABASE_DSN"},
		{"redis 口令为空", func(c *Config) { c.Redis.Password = "" }, "APP_REDIS_PASSWORD"},
		{"redis 口令为开发默认", func(c *Config) { c.Redis.Password = "ylink_redis" }, "APP_REDIS_PASSWORD"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := prodBase()
			tc.mutate(c)
			err := c.validateProd()
			require.Error(t, err, "生产模式应拒绝启动")
			assert.Contains(t, err.Error(), tc.envKey)
		})
	}
}

func TestValidateProdAcceptsRealSecrets(t *testing.T) {
	require.NoError(t, prodBase().validateProd())
}

func TestValidateProdSkipsDevelopment(t *testing.T) {
	c := prodBase()
	c.App.Env = "development"
	c.JWT.Secret = "dev-only-secret-please-override-0123456789abcdef"
	c.Database.DSN = "host=db password=ylink_root dbname=ylink"
	c.Redis.Password = "ylink_redis"
	require.NoError(t, c.validateProd())
}

func TestDSNPassword(t *testing.T) {
	cases := []struct{ dsn, want string }{
		{"host=127.0.0.1 port=5433 user=ylink password=ylink_root dbname=ylink-backend sslmode=disable TimeZone=Asia/Shanghai", "ylink_root"},
		{"password='qpass' host=db", "qpass"},
		{"postgres://ylink:p%40ss@db:5432/ylink?sslmode=disable", "p@ss"},
		{"postgres://ylink@db:5432/ylink", ""},
		{"host=db user=ylink dbname=ylink", ""},
		{"${APP_DATABASE_DSN}", ""},
	}
	for _, tc := range cases {
		if got := dsnPassword(tc.dsn); got != tc.want {
			t.Errorf("dsnPassword(%q) = %q, want %q", tc.dsn, got, tc.want)
		}
	}
}

// Load 集成：占位符 config.yaml 在生产模式拒启并指明环境变量，注入真实值后放行。
func TestLoadProductionFailFast(t *testing.T) {
	neutralizeEnv(t)
	path := writeTestConfig(t, `
app:
  env: production
database:
  dsn: "${APP_DATABASE_DSN}"
redis:
  password: "${APP_REDIS_PASSWORD}"
jwt:
  secret: "${APP_JWT_SECRET}"
`)
	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "APP_JWT_SECRET")

	prodSecretEnv(t)
	cfg, err := Load(path)
	require.NoError(t, err)
	assert.True(t, cfg.App.IsProduction())
}

// Load 集成：开发模式沿用开发默认值不受 fail-fast 影响。
func TestLoadDevelopmentUnaffected(t *testing.T) {
	neutralizeEnv(t)
	path := writeTestConfig(t, `
app:
  env: development
database:
  dsn: "host=127.0.0.1 port=5433 user=ylink password=ylink_root dbname=ylink-backend sslmode=disable"
redis:
  password: "ylink_redis"
jwt:
  secret: "dev-only-secret-please-override-0123456789abcdef"
`)
	cfg, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "ylink_root", dsnPassword(cfg.Database.DSN))
}

// APP_ENV 短名经 BindEnv 生效（部署模板/dev 脚本统一用短名，映射不到则生产标识失效）。
// 注意 viper AutomaticEnv 只覆盖文件中存在的键，故 yaml 需包含被覆盖的三类密钥键。
func TestLoadAppEnvShortName(t *testing.T) {
	neutralizeEnv(t)
	path := writeTestConfig(t, `
app:
  env: development
database:
  dsn: "${APP_DATABASE_DSN}"
redis:
  password: "${APP_REDIS_PASSWORD}"
jwt:
  secret: "${APP_JWT_SECRET}"
`)
	t.Setenv("APP_ENV", "production")
	prodSecretEnv(t)
	cfg, err := Load(path)
	require.NoError(t, err)
	assert.True(t, cfg.App.IsProduction(), "APP_ENV=production 应生效")
}
