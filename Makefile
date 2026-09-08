# YLink 本地检查入口
#
# 背景:Go 后端按项目决策不走 GitHub Actions(见 docs/backend/README.md §9),
# 推送前的后端检查统一走本 Makefile;前端检查直接用 pnpm 脚本(见 package.json)。

.PHONY: help go-vet go-test go-check lint typecheck test check

help: ## 显示帮助
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

go-vet: ## 后端 go vet ./...
	cd server && go vet ./...

go-test: ## 后端 go test ./...(全量)
	cd server && go test ./... -count=1

go-check: go-vet go-test ## 后端 vet + test 全量

lint: ## 前端 lint(含 error-toast / tokens 双源校验)
	pnpm lint

typecheck: ## 前端类型检查
	pnpm typecheck

test: ## 前端单元测试
	pnpm test

check: lint typecheck test go-check ## 前端+后端一键全量检查(提交前)
