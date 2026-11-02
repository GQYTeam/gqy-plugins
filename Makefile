# gqy-plugins：GQY 插件在线目录 + gqy-plugin CLI
#
# `make check` 是唯一验收入口，CI 用同口径的分步命令。

GO ?= go

.PHONY: help check fmt vet test build tidy clean

help: ## 列出可用目标
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-8s %s\n", $$1, $$2}'

check: ## 验收：gofmt + vet + test + build（CI 同口径）
	@test -z "$$(gofmt -l .)" || { echo "以下文件未 gofmt，请跑 make fmt："; gofmt -l .; exit 1; }
	$(GO) vet ./...
	$(GO) test ./...
	$(GO) build ./...

fmt: ## 格式化全部 Go 代码
	gofmt -w .

vet: ## 静态检查
	$(GO) vet ./...

test: ## 跑单元测试
	$(GO) test ./...

build: ## 编译全部包
	$(GO) build ./...

tidy: ## 整理 go.mod / go.sum
	$(GO) mod tidy

clean: ## 清理构建产物
	rm -rf dist coverage.out
