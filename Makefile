# gqy-plugins：GQY 插件在线目录 + gqy-plugin CLI
#
# `make check` 是唯一验收入口，CI 用同口径的分步命令。

GO ?= go
BINARY ?= dist/gqy-plugin
SHOW_DIR ?= ../gqy-plugins-ShowImages-default
EMOTE_DIR ?= ../gqy-plugins-EmotePack-default

.PHONY: help check fmt vet test build package verify sync-org web-check tidy clean

help: ## 列出可用目标
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-10s %s\n", $$1, $$2}'

check: ## 验收：gofmt + vet + test + build + web 语法 + 目录 strict 校验
	@test -z "$$(gofmt -l .)" || { echo "以下文件未 gofmt，请跑 make fmt："; gofmt -l .; exit 1; }
	$(GO) vet ./...
	$(GO) test ./...
	$(GO) build ./...
	$(MAKE) web-check
	$(MAKE) verify

fmt: ## 格式化全部 Go 代码
	gofmt -w .

vet: ## 静态检查
	$(GO) vet ./...

test: ## 跑单元测试
	$(GO) test ./...

build: ## 构建 gqy-plugin 到 dist/
	@mkdir -p dist
	$(GO) build -trimpath -o $(BINARY) ./cmd/gqy-plugin

package: build ## 打包两个带 Release 的官方插件到 dist/
	$(BINARY) package $(SHOW_DIR) --output dist/show-images-default-1.0.0.tar.gz
	$(BINARY) package $(EMOTE_DIR) --output dist/emote-pack-default-1.0.0.tar.gz

verify: ## 用 strict 口径校验 registry.json（不下载产物）
	$(GO) run ./cmd/gqy-plugin verify --registry registry.json --strict

sync-org: ## 用 GITHUB_TOKEN 自动收录组织内插件
	$(GO) run ./cmd/gqy-plugin sync-org --org GQYTeam --registry registry.json

web-check: ## 零构建站点脚本语法检查
	node --check web/app.js

tidy: ## 整理 go.mod / go.sum
	$(GO) mod tidy

clean: ## 清理构建产物
	rm -rf dist coverage.out
