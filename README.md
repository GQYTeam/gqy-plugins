# gqy-plugins

[![CI](https://github.com/GQYTeam/gqy-plugins/actions/workflows/ci.yml/badge.svg)](https://github.com/GQYTeam/gqy-plugins/actions/workflows/ci.yml)
[![Publish](https://github.com/GQYTeam/gqy-plugins/actions/workflows/publish-catalog.yml/badge.svg)](https://gqyteam.github.io/gqy-plugins/)

GQY 插件在线目录与 `gqy-plugin` CLI。

- **在线目录**：[gqyteam.github.io/gqy-plugins](https://gqyteam.github.io/gqy-plugins/)
- **目录源文件**：[`registry.json`](registry.json)（main 可审计基目录）
- **线上发布目录**：[`published:registry.json`](https://raw.githubusercontent.com/GQYTeam/gqy-plugins/published/registry.json)
- **主程序**：[GQYTeam/GQY](https://github.com/GQYTeam/GQY)

`main:registry.json` 是可审计的基础目录；`publish-catalog.yml` 叠加组织内实时插件后生成 `published:registry.json`，静态站与 CLI 默认读线上发布目录。每个安装包都带不可变 URL、SHA-256 与大小。

收录分两条线：**组织内公开插件自动收录**；**组织外在线仓库一律自动生成 PR**，不会直接写入 main。

## 快速开始

需要 Go 1.22+ 和 GQY 运行环境；当前官方插件还需要 Node 18+。

```bash
# 1. 先安装 CLI
go install github.com/GQYTeam/gqy-plugins/cmd/gqy-plugin@latest

# 2. 浏览（二进制默认在 $(go env GOPATH)/bin）
"$(go env GOPATH)/bin/gqy-plugin" list
"$(go env GOPATH)/bin/gqy-plugin" search 图片

# 3. 安装并自动通知 GQY 重新扫描、加载插件
"$(go env GOPATH)/bin/gqy-plugin" install show-images-default \
  --server http://127.0.0.1:5173 --load
```

`GQY_HOME` 默认是 `~/.gqy`。`--server http://127.0.0.1:5173 --load` 会在安装成功后调用 `POST /api/plugins/reload` 与 `POST /api/plugins/<id>/load`，卡片上的安装命令默认包含这两步。

## 当前收录

| 插件 | 版本 | 动作 | 产物 |
| --- | --- | --- | --- |
| `show-images-default` | 1.0.0 | `list` / `show` | [Release v0.1.0](https://github.com/GQYTeam/gqy-plugins/releases/tag/v0.1.0) |
| `emote-pack-default` | 1.0.0 | `list` / `show` / `match` | [Release v0.1.0](https://github.com/GQYTeam/gqy-plugins/releases/tag/v0.1.0) |
| `github-account-default` | 1.0.0 | `status` / `login` / `login-wait` / `logout` / `env` / `gh` / `git` | 组织源码归档（commit-pinned tarball） |
| `migrator-default` | 1.0.0 | `run` | 一次性插件（ONESHOT，不常驻安装） |

**收录规则：**

- `GQYTeam/gqy-plugins-*` 公开插件仓库：`publish-catalog.yml` 每 6 小时扫描一次，自动校验并生成 `published:registry.json`，同时部署 Pages。
- 组织外 GitHub 仓库：通过「插件收录申请」issue 或手动 dispatch 触发 `external-plugin-pr.yml`，自动生成 PR；必须 CI 全绿 + 一次人工评审。
- 已有同 `id@version` 的产物不可替换；想更新必须发布新版本。

## 安装语义

`gqy-plugin install` 固定执行：

1. 读取目录，解析 `id@version`；未写版本时选最新非 `yanked` 版本。
2. 下载 `.tar.gz`，边下边算 SHA-256，并核对目录登记的大小。
3. 拒绝路径穿越、绝对路径、符号链接、设备文件、超大解包。
4. 校验包根 `manifest.json` 的 `id`、`version`、`mode` 与 `protocol`。
5. 在 `$GQY_HOME/plugins/` 内 staging，最后原子 rename 到 `<id>/`；`--force` 覆盖时先备份，失败回滚。

```bash
gqy-plugin install emote-pack-default --dry-run     # 只下载校验，不落盘
gqy-plugin install emote-pack-default --force       # 覆盖已安装目录
gqy-plugin install emote-pack-default --json
```

默认**不自动加载**。安装插件 = 信任它：插件以 GQY 进程同等权限运行，能读、能写、能联网。SHA-256 只保证传输完整，不提供代码签名或沙箱。

## 目录格式

```json
{
  "schema_version": 1,
  "updated_at": "2026-11-02T21:25:00+08:00",
  "plugins": [
    {
      "id": "show-images-default",
      "name": "图片展示示例",
      "summary": "一句话简介",
      "tags": ["图片"],
      "source_type": "org",
      "repository": "https://github.com/GQYTeam/gqy-plugins-ShowImages-default",
      "versions": [
        {
          "version": "1.0.0",
          "mode": "resident",
          "protocol": "gqy-jsonrpc-v1",
          "actions": ["list", "show"],
          "url": "https://github.com/GQYTeam/gqy-plugins/releases/download/v0.1.0/show-images-default-1.0.0.tar.gz",
          "sha256": "64 位小写十六进制",
          "size": 31375314
        }
      ]
    }
  ]
}
```

完整字段、版本选择与 strict 规则见 [`docs/reference/registry-schema.md`](docs/reference/registry-schema.md)。

## 校验、收录与发布

```bash
make check
gqy-plugin verify --strict                 # 只校验目录元数据
gqy-plugin verify --artifacts              # 逐一下载产物，核对哈希、大小与 manifest
gqy-plugin package ./my-plugin --output dist/my-plugin-1.0.0.tar.gz
```

打包器是确定性的：路径排序、固定时间戳与 uid/gid、权限归一。输出 JSON 包含真实 `sha256` 与 `size`。

### 组织内：自动收录

公开仓库按标准命名为 `gqy-plugins-<Name>-default`，根目录有合法 `manifest.json` 即可被发现：

```bash
# workflow 内部等价命令
GITHUB_TOKEN=$(gh auth token) gqy-plugin sync-org --org GQYTeam --registry registry.json --json
```

同步器会读取默认分支 HEAD，下载 commit SHA 固定的 tarball，计算真实 SHA-256 / size，解析 manifest 并写入目录。已有 Release 产物的同版本条目保持不动；同版本内容不同则报告冲突并要求 bump version。工作流不写受保护的 main，而是把生成结果 fast-forward 到 `published` 分支并部署 Pages。

### 组织外：自动 PR

外部在线 GitHub 仓库使用 [「插件收录申请」](https://github.com/GQYTeam/gqy-plugins/issues/new?template=plugin_submission.yml) 提交：

1. 产物必须是同一来源仓库 `releases/download/` 下的 `.tar.gz` / `.tgz`，且版本已发布。
2. issue 或 `workflow_dispatch` 触发 `external-plugin-pr.yml`。
3. 机器人下载产物，校验 SHA-256、大小、manifest 与来源仓库，自动创建/更新 PR。
4. PR 只改 `registry.json`，必须 CI 全绿并完成一次人工评审；workflow 没有合并步骤。

外部仓库如果属于 `GQYTeam` 会被 `add-external` 拒绝，必须走组织内自动收录。

## 仓库结构

```text
.
├── registry.json              # main 可审计基目录（发布时叠加组织内插件）
├── cmd/gqy-plugin/            # CLI 子命令
├── internal/registry/         # 目录读取、校验、版本解析
├── internal/manifest/         # 包内 manifest 校验
├── internal/archive/          # 确定性打包、安全解包
├── internal/install/          # 下载、SHA-256、原子安装
├── internal/catalog/          # 组织同步、外部提交、产物校验、目录合并
├── internal/gqyapi/           # 可选 reload / load 通知
├── web/                       # 零构建静态目录站
├── docs/                      # designs / blueprint / construction / reference
└── .github/                   # CI、Pages、组织同步、外部自动 PR、Issue / PR 模板
```

## 开发

```bash
make check        # gofmt + vet + test + build + web 语法 + strict 目录校验
make test
make build        # dist/gqy-plugin
make package      # 重新打包两个带 Release 的官方插件（默认读取相邻插件仓库）
SHOW_DIR=/path/to/show EMOTE_DIR=/path/to/emote make package
make sync-org     # 用 GITHUB_TOKEN 同步组织内插件
make clean
```

`make package` 默认从组织工作区中相邻的 `gqy-plugins-ShowImages-default/` 与 `gqy-plugins-EmotePack-default/` 读取源码；单独克隆本仓库时用 `SHOW_DIR` / `EMOTE_DIR` 覆盖。

CI 在 Linux 上跑 Go 1.22 与 stable 矩阵，并额外下载全部产物做完整性验证。Pages 只发布 `web/` 与根 `registry.json`。组织同步 workflow 每 6 小时运行一次，也可手动触发。

## 文档

- 设计与决策：[`docs/designs/01-在线收录与安装.md`](docs/designs/01-在线收录与安装.md)
- 目录蓝图：[`docs/blueprint/registry.md`](docs/blueprint/registry.md)
- 安装器蓝图：[`docs/blueprint/installer.md`](docs/blueprint/installer.md)
- 静态站蓝图：[`docs/blueprint/site.md`](docs/blueprint/site.md)
- 自动收录与外部 PR：[`docs/blueprint/automation.md`](docs/blueprint/automation.md)
- 协议参考：[`docs/reference/registry-schema.md`](docs/reference/registry-schema.md)
- 施工留底：[`docs/construction/001-在线收录与安装仓库.md`](docs/construction/001-在线收录与安装仓库.md) · [002-组织内自动收录与外部自动 PR](docs/construction/002-组织内自动收录与外部自动PR.md)

## 许可与署名

本仓库按 **[PolyForm Noncommercial License 1.0.0](LICENSE)** 授权，全文见 [`LICENSE`](LICENSE)，来源与署名要求见 [`NOTICE`](NOTICE)。

- **禁止商用**：生产、部署、售卖、SaaS、内置于商业产品等任何商业目的的使用都不在授权范围内。
- **必须署名**：fork、二次开发、借鉴本仓库的代码 / 文档 / 设计，都必须标注来源与原作者，并原样保留 `LICENSE` 与 `NOTICE`。

```text
来源项目：GQYTeam/gqy-plugins
项目地址：https://github.com/GQYTeam/gqy-plugins
原作者：GQYTeam
许可证：PolyForm Noncommercial License 1.0.0
```
