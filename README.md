# gqy-plugins

[![CI](https://github.com/GQYTeam/gqy-plugins/actions/workflows/ci.yml/badge.svg)](https://github.com/GQYTeam/gqy-plugins/actions/workflows/ci.yml)
[![Pages](https://github.com/GQYTeam/gqy-plugins/actions/workflows/pages.yml/badge.svg)](https://gqyteam.github.io/gqy-plugins/)

GQY 插件在线目录与 `gqy-plugin` CLI。

- **在线目录**：[gqyteam.github.io/gqy-plugins](https://gqyteam.github.io/gqy-plugins/)
- **目录源文件**：[`registry.json`](registry.json)
- **主程序**：[GQYTeam/GQY](https://github.com/GQYTeam/GQY)

目录是唯一事实来源。静态站点和 CLI 都读同一份 `registry.json`，每个安装包都带不可变 Release URL、SHA-256 与大小。

## 快速开始

需要 Go 1.22+ 和 GQY 运行环境；当前官方插件还需要 Node 18+。

```bash
# 浏览
go run github.com/GQYTeam/gqy-plugins/cmd/gqy-plugin@latest list
go run github.com/GQYTeam/gqy-plugins/cmd/gqy-plugin@latest search 图片
go run github.com/GQYTeam/gqy-plugins/cmd/gqy-plugin@latest info show-images-default

# 安装到 $GQY_HOME/plugins/<id>/
go run github.com/GQYTeam/gqy-plugins/cmd/gqy-plugin@latest install show-images-default

# 让本机 GQY 重扫；只有显式 --load 才启动插件
go run github.com/GQYTeam/gqy-plugins/cmd/gqy-plugin@latest install show-images-default \
  --server http://127.0.0.1:5173 --load
```

也可以先安装 CLI：

```bash
go install github.com/GQYTeam/gqy-plugins/cmd/gqy-plugin@latest
gqy-plugin install emote-pack-default
```

`GQY_HOME` 默认是 `~/.gqy`。安装后的下一步是 `POST /api/plugins/reload`，再 `POST /api/plugins/<id>/load`；CLI 的 `--server` 与 `--load` 只是帮你调用这两个接口。

## 当前收录

| 插件 | 版本 | 动作 | 产物 |
| --- | --- | --- | --- |
| `show-images-default` | 1.0.0 | `list` / `show` | [Release v0.1.0](https://github.com/GQYTeam/gqy-plugins/releases/tag/v0.1.0) |
| `emote-pack-default` | 1.0.0 | `list` / `show` / `match` | [Release v0.1.0](https://github.com/GQYTeam/gqy-plugins/releases/tag/v0.1.0) |

新增条目必须先有真实可下载产物，再登记真实 `sha256` 与 `size`；禁止占位值。流程见下文。

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

## 校验与发布

```bash
make check
gqy-plugin verify --strict                 # 只校验目录元数据
gqy-plugin verify --artifacts              # 逐一下载产物，核对哈希、大小与 manifest
gqy-plugin package ./my-plugin --output dist/my-plugin-1.0.0.tar.gz
```

打包器是确定性的：路径排序、固定时间戳与 uid/gid、权限归一。输出 JSON 包含真实 `sha256` 与 `size`，直接用于目录条目。

发布一个新版本：

1. 在插件仓库完成自检并打 tag。
2. 用 `gqy-plugin package` 生成 `.tar.gz`，把产物上传到本仓库的 GitHub Release。
3. 在 `registry.json` 增加 `version` 条目，填真实 URL、SHA-256、大小、协议与动作。
4. 更新 `updated_at`，提 PR；CI 会跑 strict 校验，合并后再跑产物下载校验。
5. 下载产物并逐项验证：

```bash
gqy-plugin verify --strict --artifacts
```

## 仓库结构

```text
.
├── registry.json              # 在线目录唯一事实来源
├── cmd/gqy-plugin/            # CLI 子命令
├── internal/registry/         # 目录读取、校验、版本解析
├── internal/manifest/         # 包内 manifest 校验
├── internal/archive/          # 确定性打包、安全解包
├── internal/install/          # 下载、SHA-256、原子安装
├── internal/gqyapi/           # 可选 reload / load 通知
├── web/                       # 零构建静态目录站
├── docs/                      # designs / blueprint / construction / reference
└── .github/                   # CI、Pages、Issue / PR 模板
```

## 开发

```bash
make check        # gofmt + vet + test + build + web 语法 + strict 目录校验
make test
make build        # dist/gqy-plugin
make package      # 重新打包两个官方插件
make clean
```

CI 在 Linux 上跑 Go 1.22 与 stable 矩阵，并额外下载 Release 产物做完整性验证。Pages 只发布 `web/` 与根 `registry.json`。

## 文档

- 设计与决策：[`docs/designs/01-在线收录与安装.md`](docs/designs/01-在线收录与安装.md)
- 目录蓝图：[`docs/blueprint/registry.md`](docs/blueprint/registry.md)
- 安装器蓝图：[`docs/blueprint/installer.md`](docs/blueprint/installer.md)
- 静态站蓝图：[`docs/blueprint/site.md`](docs/blueprint/site.md)
- 协议参考：[`docs/reference/registry-schema.md`](docs/reference/registry-schema.md)
- 施工留底：[`docs/construction/001-在线收录与安装仓库.md`](docs/construction/001-在线收录与安装仓库.md)

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
