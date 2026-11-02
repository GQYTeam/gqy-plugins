# 变更记录

格式参照 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [0.2.0] - 2026-11-02

### Added

- 目录新增 `source_type`（`org` / `external`）与 `repository` 来源字段，站点展示“组织内自动收录 / 外部 PR 收录”。
- `gqy-plugin sync-org`：扫描 `GQYTeam/gqy-plugins-*` 公开仓库，使用默认分支 HEAD 的 commit-pinned tarball，计算 SHA-256 / size 并自动写目录。
- `gqy-plugin add-external`：解析插件收录 issue，校验同仓库 GitHub Release 产物，自动写入待 PR 的外部条目。
- `.github/workflows/sync-org-plugins.yml`：每 6 小时自动收录组织内插件，直接提交 main，并显式触发 CI 与 Pages。
- `.github/workflows/external-plugin-pr.yml`：issue / dispatch 触发，自动创建或更新外部插件 PR，没有合并步骤。
- `internal/catalog/`：GitHub 组织发现、HEAD tarball、产物下载解包、来源校验、同版本冲突保护与目录原子写入。
- 目录新增 `github-account-default@1.0.0`，由组织内自动收录使用 commit-pinned tarball。

### Changed

- 组织内收录不再需要 PR；组织外收录一律走自动 PR。
- `main` 保护从经典 branch protection 迁移为 ruleset：保留一次评审与两个 Go check；管理员和 `github-actions[bot]` 可 bypass，仅供同步 workflow 使用。
- README、设计、蓝图、协议参考、Issue 模板、CONTRIBUTING 与 Makefile 对齐新收录策略。

### Fixed

- 安全解包接受 GitHub tarball 的 PAX 扩展头（`pax_global_header`），仍拒绝符号链接、设备文件与越界路径。

## [0.1.0] - 2026-11-02

### Added

- `registry.json` 收录首批官方插件：`show-images-default@1.0.0` 与 `emote-pack-default@1.0.0`，指向 `v0.1.0` Release 产物并带真实 SHA-256 与大小。
- `gqy-plugin` CLI：`list / search / info / install / verify / package / version`，支持 JSON 输出、`--dry-run`、`--force` 与可选 `--server` / `--load`。
- `internal/registry`：目录读取（HTTPS / `file://` / 本地路径）、严格字段校验、未知字段拒绝、最新版选择、yanked 处理与版本比较。
- `internal/manifest`：安装前校验包内 P3 核心字段。
- `internal/archive`：确定性 tar.gz 打包、安全解包、路径穿越与链接拒绝、上限控制。
- `internal/install`：下载、边下边算 SHA-256、大小核对、staging 与原子 rename、force 备份回滚、产物完整校验。
- `internal/gqyapi`：安装后通知 `/api/plugins/reload`，可选 `/api/plugins/{id}/load`。
- `web/`：零构建插件目录站，支持搜索、版本、动作、大小与复制安装命令。
- `docs/`：designs / blueprint / construction / reference 四层文档，记录 REG1–REG10 决策与验收。
- `.github/workflows/pages.yml`：GitHub Pages 发布工作流。
- CI 增加 `go test -race`、Web JS 语法检查、strict 目录校验与 Release 产物逐项下载校验。

### Changed

- `Makefile` 增加 `build / package / verify / web-check`，`make check` 与 CI 同口径。
- README 从骨架状态更新为可用状态，补充快速开始、安装语义、发布流程与信任边界。
- CLI 参数解析支持 flag 出现在位置参数之后。

### Security

- 安装器在解包前完成 SHA-256 与大小校验。
- 拒绝 tar 路径穿越、绝对路径、符号链接、设备文件、超量文件与超大解压。
- force 覆盖使用同盘备份与失败回滚，正常完成后清理备份。
- 文档明确：SHA-256 不是签名、安装不是沙箱、插件以 GQY 同等权限运行。
