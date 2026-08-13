// Package repo 提供 GORM 数据访问层：DB 初始化与各实体 CRUD/查询构造（不含业务）。
package repo

import (
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"ylink-backend/internal/config"
)

// newLogger 返回 GORM 日志器：Warn 级别、慢查询阈值 200ms，关闭 ANSI 颜色
// （日志可能被重定向到文件，颜色转义序列会造成乱码）。
// IgnoreRecordNotFoundError 保持 GORM Default 日志器行为（true）：记录未找到属正常 404，不应刷 Error 日志。
func newLogger() gormlogger.Interface {
	return gormlogger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		gormlogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  gormlogger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)
}

// NewDB 初始化 GORM 连接；慢查询（>200ms）与错误写入日志。
func NewDB(cfg config.DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{
		Logger:  newLogger(),
		NowFunc: func() time.Time { return time.Now() },
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpen)
	sqlDB.SetMaxIdleConns(cfg.MaxIdle)
	sqlDB.SetConnMaxLifetime(time.Hour)
	return db, nil
}

// Ping 探测数据库连通（供 /readyz）。
func Ping(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
