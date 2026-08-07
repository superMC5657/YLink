//go:build tools

// tools.go 固定开发期 CLI 工具依赖(供 Makefile 的 go run 使用):
//   - golang-migrate: 执行 migrations/ 下的数据库迁移
package main

import _ "github.com/golang-migrate/migrate/v4/cmd/migrate"
