# 变更记录

格式参照 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

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
