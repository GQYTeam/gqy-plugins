# gqy-plugins

[![CI](https://github.com/GQYTeam/gqy-plugins/actions/workflows/ci.yml/badge.svg)](https://github.com/GQYTeam/gqy-plugins/actions/workflows/ci.yml)

GQY 插件在线目录与 `gqy-plugin` CLI。

本仓库做两件事：

1. **插件在线目录的唯一事实来源**：仓库根的 [`registry.json`](registry.json) 是官方目录，静态站点与 `gqy-plugin` CLI 只读这一份数据，不再各自维护列表。
2. **目录工具的 Go 实现**：`internal/registry` 负责读取与校验 `registry.json`（支持 https / `file://` / 本地路径三种来源）。

> **当前状态**：目录格式、校验逻辑与 `registry.json` 空目录基线已落地；**目录条目暂时为空**——不预置占位条目，只登记真正发布过产物、带可下载地址与真实 sha256 的插件。
> `gqy-plugin` CLI、安装器与静态站点**尚未实现**，见[路线图](#路线图)。

`internal/registry.DefaultURL` 指向 `https://raw.githubusercontent.com/GQYTeam/gqy-plugins/main/registry.json`，即本仓库 `main` 分支根目录的这份文件。

## 目录结构

```text
.
├── registry.json                 # 官方插件目录（唯一事实来源）
├── internal/registry/            # 目录读取与校验
│   ├── registry.go               # Load / Parse / Validate / ResolveURL
│   └── registry_test.go          # 读生产目录文件 + 各类拒绝路径
├── docs/                         # 文档索引（见 docs/README.md）
├── .github/                      # CI、Issue / PR 模板、dependabot
├── Makefile                      # make check = gofmt + vet + test + build
├── .golangci.yml                 # golangci-lint v2 配置（本地可选）
├── go.mod                        # module github.com/GQYTeam/gqy-plugins（go 1.22）
└── LICENSE / NOTICE / README.md / CONTRIBUTING.md / SECURITY.md / CHANGELOG.md
```

`cmd/`（CLI）、`internal/install`（下载安装）、`web/`（静态站点）尚未开工，所以没有对应目录，规划见路线图。

## registry.json schema

顶层 `Document`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `schema_version` | int | 是 | 目录格式版本，当前固定为 `1`（`SchemaVersion`） |
| `updated_at` | string | 是 | RFC3339 时间戳，目录最后一次更新时间 |
| `plugins` | array | 是 | 插件条目数组，暂时为空也要写 `[]`，不能省略 |
| `base` | — | — | 运行时字段（Go 里小写、不参与 JSON），由 `Load` / `ParseJSON` 记下来源目录，供 `ResolveURL` 解析相对地址 |

`plugins[]`（`Plugin`）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | string | 是 | 插件唯一标识，匹配 `^[a-z0-9][a-z0-9_-]{0,31}$`，全目录唯一 |
| `name` | string | 是 | 展示名，非空 |
| `summary` | string | 是 | 一句话简介，非空 |
| `description` | string | 否 | 长描述 |
| `homepage` | string | 否 | 插件主页，http(s) 绝对地址 |
| `tags` | string[] | 否 | 分类标签，元素非空 |
| `versions` | array | 是 | 版本列表，至少一项；版本号在同一插件内唯一 |

`plugins[].versions[]`（`Version`）：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `version` | string | 是 | 匹配 `^\d{1,4}\.\d{1,4}\.\d{1,4}$`，如 `1.0.0` |
| `channel` | string | 否 | 渠道标记（如 `stable` / `beta`），自由文本 |
| `published_at` | string | 否 | RFC3339 时间戳 |
| `mode` | string | 否 | 插件运行模式标记，自由文本 |
| `protocol` | string | 否 | 插件协议版本标记，自由文本 |
| `actions` | string[] | 否 | 可执行动作列表，元素非空 |
| `provides` | string[] | 否 | 提供的能力列表，元素非空 |
| `requires` | string[] | 否 | 依赖的能力列表，元素非空 |
| `url` | string | 是 | 产物下载地址；默认（非 strict）允许相对地址与 http(s) 绝对地址 |
| `sha256` | string | 是 | 产物 sha256，匹配 `^[a-f0-9]{64}$`（64 位小写十六进制） |
| `size` | int64 | 否 | 产物体积（字节），不能为负 |
| `yanked` | bool | 否 | 标记该版本已撤回；保留历史记录，不从数组里删 |

### 校验规则

`Parse` / `Load` 会做过 `Validate(false)`，发布目录可以用 `Validate(true)` 走 strict 模式：

- **未知字段直接拒绝**（`DisallowUnknownFields`）：字段名拼错不会被静默忽略，解析即报错。
- 顶层 `schema_version` 必须等于 `1`；`updated_at` 与 `published_at` 必须是 RFC3339；`plugins` 必须是数组。
- 插件 `id` 必须匹配 `^[a-z0-9][a-z0-9_-]{0,31}$`，且**全目录唯一**；`name`、`summary` 不能为空。
- `homepage` 若填写必须是 http(s) 绝对地址；`tags` 元素不能为空字符串。
- 每个插件至少一个版本；`version` 必须匹配 `^\d{1,4}\.\d{1,4}\.\d{1,4}$`，同一插件内**版本号唯一**。
- `url` 不能为空；strict 模式只接受 **https 绝对地址**（非 strict 允许 `http(s)` 绝对地址或相对地址，相对地址按目录来源解析）。
- `sha256` 必须是 64 位小写十六进制；`size` 不能为负；`actions` / `provides` / `requires` 元素不能为空字符串。
- 上述规则的真实实现见 [`internal/registry/registry.go`](internal/registry/registry.go)，文档与其不一致时以代码为准，并顺手更正本文。

## 本地验收

```bash
make check   # gofmt 检查 + go vet ./... + go test ./... + go build ./...
```

其它目标：

| 目标 | 作用 |
| --- | --- |
| `make fmt` | `gofmt -w .` 格式化全部 Go 代码 |
| `make vet` | `go vet ./...` |
| `make test` | `go test ./...` |
| `make build` | `go build ./...` |
| `make tidy` | `go mod tidy` |
| `make clean` | 清理 `dist/`、`coverage.out` |
| `make help` | 列出全部目标 |

CI（[`.github/workflows/ci.yml`](.github/workflows/ci.yml)）在 `push: main` 与 PR 上跑：Go `1.22.x` + `stable` 矩阵，依次检查 gofmt、`go vet`、`go test`、`go build`。

## 新增插件条目

目录**不预置占位数据**。追加条目的前提是插件产物真的已经发布：

1. 插件产物（zip / tar）发布到可公开下载的 **https** 地址；
2. 对产物算真实 sha256（例：`shasum -a 256 gqy-plugins-<Name>-default-1.0.0.zip`），记下字节数；
3. 在 `registry.json` 的 `plugins` 里追加条目，`url` / `sha256` / `size` 填**真实值**；
4. 把 `updated_at` 更新为当前 RFC3339 时间；
5. 跑 `make check`（测试会读仓库根 `registry.json` 做真实校验），然后按 [`CONTRIBUTING.md`](CONTRIBUTING.md) 提 PR，走一次评审合入。

> 禁止为了占位写假 `url`、假 `sha256`、假 `size`；没有可下载产物就不加条目。

## 路线图

- [x] 目录格式与校验（`internal/registry`：`Load` / `Parse` / `Validate` / `ResolveURL`）
- [x] `registry.json` 空目录基线 + `make check` + CI
- [ ] `gqy-plugin` CLI：`search` / `list` / `install` / `update` / `remove`（未实现，开工时新建 `cmd/gqy-plugin/`）
- [ ] `internal/install`：下载、sha256 校验、安装落盘（未实现）
- [ ] 静态站点 `web/`：渲染目录、搜索与安装指引（未实现）
- [ ] 目录签名与镜像源（未开始，先解决可信来源问题）

## 许可与署名

本仓库按 **[PolyForm Noncommercial License 1.0.0](LICENSE)** 授权，全文见 [`LICENSE`](LICENSE)，来源与署名要求见 [`NOTICE`](NOTICE)。

- **禁止商用**：生产、部署、售卖、SaaS、内置于商业产品等任何商业目的的使用都不在授权范围内。
- **必须署名**：fork、二次开发、借鉴本仓库的代码 / 文档 / 设计，都必须标注来源与原作者，并原样保留 `LICENSE` 与 `NOTICE`。

衍生项目请在 README（或同等声明文件）里原样写清：

```text
来源项目：GQYTeam/gqy-plugins
项目地址：https://github.com/GQYTeam/gqy-plugins
原作者：GQYTeam
许可证：PolyForm Noncommercial License 1.0.0
```

本仓库目前只依赖 Go 标准库，无第三方代码或素材；一旦引入，按组织标准把许可与出处记在 `docs/third-party/`。

## 相关文档

[`docs/README.md`](docs/README.md) · [`CONTRIBUTING.md`](CONTRIBUTING.md) · [`SECURITY.md`](SECURITY.md) · [`CHANGELOG.md`](CHANGELOG.md)
