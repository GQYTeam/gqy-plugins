# 变更记录

格式参照 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### Added

- `registry.json`：仓库根的官方插件目录（`schema_version` 为 1，`plugins` 暂为空数组；条目在插件真的发布产物后再追加）。
- `internal/registry`：`Load` / `Parse` / `Validate` 与目录校验（id、semver、sha256、RFC3339、strict HTTPS、未知字段拒绝、重复 id / 版本拒绝）。
- `internal/registry/registry_test.go`：读生产目录文件与各类拒绝路径的单元测试。
- 仓库标准基线：`LICENSE`（PolyForm Noncommercial 1.0.0 + Required Notice）、`NOTICE`、`CONTRIBUTING.md`、`SECURITY.md`、`.editorconfig`、`.gitignore`、`Makefile`（`make check`）、`.golangci.yml`、`docs/README.md`。
- `.github/`：CI（Go 1.22 + stable 矩阵，gofmt / vet / test / build）、Issue 与 PR 模板、dependabot（gomod + github-actions）。

### Changed

- `internal/registry/registry.go` 补齐 `readSource`（https / file:// / 本地路径）与 `validatePlugin` 等缺失实现，并按 gofmt 重排。

### Fixed

- 修复 `internal/registry` 无法编译：`readSource`、`validatePlugin` 此前只有调用没有定义，import 块与函数体也未 gofmt。
