// Package logger 封装 zap 结构化日志，JSON 格式按天切割（lumberjack），同时输出 stdout。
package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"ylink/internal/config"
)

var std *zap.Logger

// Init 初始化全局日志器。dir 为空时仅输出 stdout。
func Init(cfg config.LogConfig) error {
	level := zapcore.InfoLevel
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = zapcore.DebugLevel
	case "warn", "warning":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	}
	enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stack",
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
	})

	cores := []zapcore.Core{zapcore.NewCore(enc, zapcore.AddSync(os.Stdout), level)}
	if cfg.Dir != "" {
		lj := &lumberjack.Logger{
			Filename:   strings.TrimRight(cfg.Dir, "/\\") + "/app.log",
			MaxSize:    100, // MB
			MaxAge:     14,  // 天
			MaxBackups: 7,
			LocalTime:  true,
			Compress:   true,
		}
		cores = append(cores, zapcore.NewCore(enc, zapcore.AddSync(lj), level))
	}

	std = zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddCallerSkip(0))
	return nil
}

// L 返回全局 logger。
func L() *zap.Logger { return std }

// Nop 供测试使用。
func Nop() { std = zap.NewNop() }
