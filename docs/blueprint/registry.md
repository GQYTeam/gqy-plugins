# registry · 在线目录蓝图

## 1. 状态

已实现（2026-11-02）。对应设计：REG1、REG2、REG3、REG4、REG9。

## 2. 文件与落点

| 位置 | 职责 |
| --- | --- |
| `registry.json` | main 上的可审计基础目录；发布时叠加组织内实时插件 |
| `internal/registry/registry.go` | 读取（HTTPS / `file://` / 本地路径）、JSON 解析、字段校验、相对 URL 解析 |
| `internal/registry/resolve.go` | `Find`、版本选择、三位版本比较 |
| `internal/registry/registry_test.go` | 生产目录真实校验与拒绝路径 |
| `internal/registry/resolve_test.go` | 最新版、yanked、版本比较 |
| `published:registry.json` | `publish-catalog.yml` 生成的线上目录，CLI / 浏览器默认读取 |

## 3. 顶层结构

```json
{
  "schema_version": 1,
  "updated_at": "2026-11-02T21:25:00+08:00",
  "plugins": []
}
```

- `schema_version`：必须是整数 `1`。
- `updated_at`：RFC3339；每次目录内容变化都要更新。
- `plugins`：数组，可以为空，但字段不能省略。

## 4. 插件与版本字段

插件：

| 字段 | 必填 | 校验 |
| --- | --- | --- |
| `id` | 是 | `^[a-z0-9][a-z0-9_-]{0,31}$`，全目录唯一 |
| `name` | 是 | 非空，展示名 |
| `summary` | 是 | 非空，一句话简介 |
| `description` | 否 | 长描述 |
| `homepage` | 否 | http(s) 绝对地址；strict 模式要求 https |
| `tags` | 否 | 字符串数组，元素非空 |
| `source_type` | 否 | `org` / `external`；出现时必须带 `repository` |
| `repository` | 否 | 来源仓库 http(s) 绝对地址；strict 要求 https |
| `versions` | 是 | 至少一项，版本号不重复 |

版本：

| 字段 | 必填 | 校验 |
| --- | --- | --- |
| `version` | 是 | `^\d{1,4}\.\d{1,4}\.\d{1,4}$` |
| `channel` | 否 | 渠道自由文本，当前固定 `stable` |
| `published_at` | 否 | RFC3339 |
| `mode` | 否 | `resident` / `oneshot`；`install` 拒绝 oneshot |
| `protocol` | 否 | `gqy-jsonrpc-v1` / `mcp` / `unix` / `tcp` |
| `actions` | 否 | 动作名数组，元素匹配 `^[a-zA-Z][a-zA-Z0-9_-]{0,63}$` |
| `provides` / `requires` | 否 | 服务名数组，元素匹配 `^[a-z][a-z0-9_.-]{0,63}$` |
| `url` | 是 | 产物地址；默认相对或 http(s)，strict 只接受 https 绝对地址 |
| `sha256` | 是 | 64 位小写十六进制 |
| `size` | 否 | 字节数，大于等于 0，可被安装器严格比对 |
| `yanked` | 否 | 撤回标记；保留历史，不参与 latest 选择 |

## 5. 版本选择

1. 显式 `id@version`：精确匹配；`yanked:true` 拒绝。
2. 只给 `id`：过滤掉 `yanked`，按 `MAJOR.MINOR.PATCH` 数值排序取最大。
3. 没有任何可安装版本：报错。

版本比较不做预发布后缀；目录 schema 当前只接受三位数字版本。

## 6. URL 解析

`registry.Load` 记录目录来源的 base：

- HTTPS 来源：相对 `url` 按来源 URL 所在目录解析。
- 本地路径 / `file://`：相对 `url` 按目录文件所在目录解析。
- 绝对 http(s) URL：原样使用。

发布目录中的版本 URL 必须是 HTTPS 绝对地址：外部条目来自 `https://github.com/<owner>/<repo>/releases/download/...`，组织内无 Release 时来自 `https://api.github.com/repos/GQYTeam/<repo>/tarball/<sha>`。CI 对 main 目录执行 strict 校验。

## 7. 校验时机

| 时机 | 行为 |
| --- | --- |
| `registry.Parse` | 无条件基础校验与未知字段拒绝 |
| `registry.Load` | 读取来源后执行 `Parse` |
| `gqy-plugin verify --strict` | 额外要求 HTTPS 绝对产物地址 |
| `gqy-plugin verify --artifacts` | 逐版本下载产物，核对 sha256、size、包内 manifest |
| `make check` / CI | strict 元数据校验，不下载大产物 |

## 8. 来源分层

- `source_type=org`：由 `gqy-plugin sync-org` 扫描 `GQYTeam/gqy-plugins-*` 公开仓库生成，随后由 `publish-catalog.yml` 写到 `published` 分支。
- `source_type=external`：由 `external-plugin-pr.yml` 验证外部 GitHub Release 后生成 PR，人工评审合入。
- 发现同 id 同版本但 URL / SHA-256 不同：保留旧产物并报告冲突，必须发布新版本。

## 9. 当前官方条目

`registry.json` 当前快照包含四个组织内插件：

- `emote-pack-default@1.0.0`
- `github-account-default@1.0.0`（commit-pinned tarball）
- `migrator-default@1.0.0`（commit-pinned tarball，`mode: oneshot`）
- `show-images-default@1.0.0`

前两个仓库的已有 Release 产物保持不变；没有 Release 的组织插件使用 commit-pinned tarball。新增条目禁止占位值。
