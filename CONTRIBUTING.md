# 贡献指南

本仓库属于 GQYTeam，遵循组织的生产项目基线。改代码前先读 `docs/`；文档与代码不一致时以代码为准，并顺手更正文档。

## 分支与 PR

- **一个改动一个分支一个 PR。** 分支名：`feat/<主题>`、`fix/<主题>`、`docs/<主题>`、`chore/<主题>`。
- 默认分支 `main` 受保护：合并前必须 CI 全绿 + 一次评审。
- PR 描述写清三件事：**动机**、**影响面**、**怎么验的**。
- 不要在 PR 里混无关改动。

## 本地验收

```bash
make check
```

改动目录条目时还要跑：

```bash
go run ./cmd/gqy-plugin verify --registry registry.json --strict
go run ./cmd/gqy-plugin verify --registry registry.json --strict --artifacts
```

## 收录插件

### 组织内插件：自动收录

`GQYTeam` 下的公开插件仓库遵循：

- 仓库名 `gqy-plugins-<Name>-default`；
- 根目录有合法 `manifest.json`；
- 默认分支 HEAD 可构建；插件版本变化必须修改 `manifest.json` 的 `version`；
- 同 `id@version` 已收录后不再覆盖产物，修改内容请发新版本。

满足以上条件后，`publish-catalog.yml` 每 6 小时自动发现并生成 `published:registry.json` 与 Pages 站点。维护者也可手动运行：

```bash
GITHUB_TOKEN=$(gh auth token) go run ./cmd/gqy-plugin sync-org --org GQYTeam
```

组织内插件不通过外部 PR 通道。

### 组织外插件：自动 PR

组织外 GitHub 仓库必须通过 [「插件收录申请」](https://github.com/GQYTeam/gqy-plugins/issues/new?template=plugin_submission.yml) 提交，或由维护者手动触发 `External plugin PR` workflow。

要求：

1. 来源仓库必须是公开 GitHub 仓库，owner 不能是 `GQYTeam`。
2. 产物必须是同一仓库 `releases/download/` 下的 `.tar.gz` / `.tgz`，版本已发布且不可变。
3. 包根或唯一顶层目录必须有合法 `manifest.json`。
4. issue 中填写的 SHA-256 与大小必须与真实产物一致。
5. workflow 只创建/更新 PR，不自动合并；合入前必须 CI 全绿 + 一次人工评审。

禁止同时用组织内和外部通道收录同一个插件；`gqy-plugin add-external` 会拒绝组织仓库。

## 提交信息
## 提交信息

- 主题行用祈使句、不超过 72 字符，例如 `feat: 收录表情包插件 1.0.0`。
- 正文写动机与影响面；关联 issue 用 `Closes #123`。
- 不提交构建产物、Release 二进制、临时文件与密钥。Release 产物通过 GitHub Release 上传。

## 规范

1. **先改文档再改代码。** 行为变了，同一 PR 里更新 `docs/` 与 `CHANGELOG.md`。
2. **安装器必须保持零第三方依赖**，除非先改设计并说明取舍。
3. **测试走生产路径。** 安全边界（路径穿越、哈希、原子覆盖）必须有测试。
4. **许可证。** 贡献即同意按本仓库 `LICENSE`（PolyForm Noncommercial License 1.0.0）分发，并保留 `NOTICE` 的署名要求。
