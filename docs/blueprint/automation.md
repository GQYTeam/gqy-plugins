# automation · 组织自动收录与外部自动 PR

## 1. 状态

已实现（2026-11-02）。对应设计：REG11、REG12、REG13。

## 2. 代码与工作流

| 位置 | 职责 |
| --- | --- |
| `cmd/gqy-plugin/sync_org.go` | 手动/CI 触发组织插件同步 |
| `cmd/gqy-plugin/add_external.go` | 解析 issue 表单、校验外部产物、写入候选目录 |
| `internal/catalog/github.go` | GitHub 组织仓库、HEAD commit、tarball URL |
| `internal/catalog/sync.go` | 组织插件发现、版本追加、冲突报告、原子写目录 |
| `internal/catalog/external.go` | 外部仓库 / Release URL 约束与来源类型校验 |
| `internal/catalog/inspect.go` | 下载、SHA-256、解包、读取 manifest |
| `internal/catalog/upsert.go` | 目录条目合并、同版本产物保护 |
| `.github/workflows/publish-catalog.yml` | 每 6 小时 + main 更新 + 手动触发：同步组织插件、写 `published` 分支并部署 Pages |
| `.github/workflows/external-plugin-pr.yml` | issue / 手动触发外部自动 PR |

## 3. 组织内自动收录

触发：

- `schedule: 17 */6 * * *`
- `workflow_dispatch`

流程：

1. 用 `GITHUB_TOKEN` 列出 `GQYTeam` 下公开、非 archived、非 fork、名称以 `gqy-plugins-` 开头的仓库。
2. 取默认分支 HEAD commit SHA，构造 commit-pinned tarball URL。
3. 下载并解包；校验 manifest；计算真实 SHA-256 与 size。
4. `source_type=org`，`repository=https://github.com/GQYTeam/<repo>`。
5. 同版本已有且产物不同：保留旧条目并报告冲突，不覆盖。
6. 有新版本或元数据变化：写本地 `registry.json`，strict 校验。
7. 把生成文件提交到 `published` 分支（只做 fast-forward push），并部署 Pages；CLI 默认读取 `published:registry.json`。

组织内插件不需要 PR，也不写受保护的 `main`。main 只保留可审计的基础目录与外部 PR 条目；发布文件由 workflow 从 main + 实时组织仓库生成。

## 4. 外部自动 PR

触发：

- 仓库 issue 使用「插件收录申请」模板（标题含 `[Plugin]` 或 label `plugin-submission`）。
- 维护者通过 `workflow_dispatch` 提供仓库 URL 与 Release URL。

校验：

- 来源仓库必须是 `https://github.com/<owner>/<repo>`，且 owner 不能是 `GQYTeam`。
- 产物必须是同一仓库 `releases/download/` 下的 `.tar.gz` / `.tgz` 资产。
- 下载后校验真实 SHA-256，并与 issue 里填写的 SHA-256 / size 对比。
- 解包后校验 manifest id / version，自动写入 `source_type=external`、`repository`、`version`、`sha256`、`size`。

写目录方式：

- workflow 在临时分支 `auto/external-plugin-<issue-number>` 上修改 `registry.json`。
- 使用 `gh pr create` 创建 PR（若已存在则更新同一 PR）。
- 显式 `workflow_dispatch` 触发 CI，因为 bot 创建的分支 push 不会自动触发其它 workflow。
- 不包含自动合并步骤；外部 PR 必须通过 CI 且完成一次人工评审。

## 5. 目录字段

```json
{
  "source_type": "org",
  "repository": "https://github.com/GQYTeam/gqy-plugins-ShowImages-default"
}
```

- `source_type`：`org` / `external`；与 `repository` 同时出现。
- `repository`：来源仓库 HTTPS 地址。
- 外部条目只能由 `add-external` 写入；组织内同步遇到来源类型冲突直接报错跳过。

## 6. 分支保护

`main protection` ruleset：

- 禁止删除、禁止强推。
- 要求一次 PR 评审，评审线程必须解决，过期评审在 push 后 dismiss。
- 要求 `Check (Go 1.22.x)` 与 `Check (Go stable)` 通过。
- bypass actors：仅 RepositoryRole admin（应急）。

`published protection` ruleset：

- 禁止删除、禁止强推。
- 不要求 PR；允许 `publish-catalog.yml` 以正常 fast-forward 提交生成目录。
- 目录文件只由工作流生成，不接收人工 PR。

外部 PR workflow 没有 merge / auto-merge 步骤。

## 7. 手工命令

```bash
# 组织内同步（需要 GITHUB_TOKEN）
GITHUB_TOKEN=$(gh auth token) go run ./cmd/gqy-plugin sync-org --org GQYTeam

# 校验外部候选（不执行插件代码，只下载解析）
go run ./cmd/gqy-plugin add-external \
  --repository https://github.com/<owner>/<repo> \
  --artifact-url https://github.com/<owner>/<repo>/releases/download/<tag>/<file>.tar.gz \
  --registry registry.json --json
```

## 8. 验收

- 组织同步在同一份 HEAD 上重复跑，不改写 `registry.json`。
- 无 Release 的组织插件使用 commit-pinned tarball。
- 篡改产物、错哈希、错 size、错误来源、组织内仓库冒充外部均被拒绝。
- 外部 issue 触发只生成/更新 PR，main 不出现直接提交。
- `publish-catalog.yml` 能生成 `published:registry.json` 并部署 Pages；`published` 禁止强推和删除。
- CLI 默认读取 `https://raw.githubusercontent.com/GQYTeam/gqy-plugins/published/registry.json`。
