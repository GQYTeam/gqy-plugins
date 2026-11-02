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

## 收录新插件

1. **先发布产物。** 在插件仓库自检并打 tag，用 `gqy-plugin package <plugin-dir> --output dist/<id>-<version>.tar.gz --json` 生成确定性 `.tar.gz`，拿到真实 `sha256` 与 `size`。
2. **上传 Release。** 把产物作为本仓库 GitHub Release 的资产上传。目录里的 `url` 必须指向不可变 Release 资产，不能指向分支归档或会变的下载地址。
3. **补目录。** 在 `registry.json` 的对应插件或新插件条目写入真实元数据；禁止占位 `url` / `sha256` / `size`。同步更新 `updated_at`。
4. **跑校验。** `make check` 与 `verify --strict --artifacts` 都通过。
5. **提 PR。** PR 里贴产物下载地址、SHA-256、验证命令输出，并确认插件来源与许可证。

### 版本与撤回

- 版本固定 `MAJOR.MINOR.PATCH`。
- 同版本号不可覆盖发布；修 bug 必须发新版本。
- 发现严重问题时把旧版本标 `"yanked": true`，保留历史记录，不再参与 latest 选择。
- `id` 全局唯一，改名视为新插件。

### 一次性插件

目录支持 `mode: "oneshot"` 作为未来元数据。当前 `gqy-plugin install` 只安装常驻插件（`mode` 缺省或 `resident`）；一次性插件需要显式运行，不走安装命令。

## 提交信息

- 主题行用祈使句、不超过 72 字符，例如 `feat: 收录表情包插件 1.0.0`。
- 正文写动机与影响面；关联 issue 用 `Closes #123`。
- 不提交构建产物、Release 二进制、临时文件与密钥。Release 产物通过 GitHub Release 上传。

## 规范

1. **先改文档再改代码。** 行为变了，同一 PR 里更新 `docs/` 与 `CHANGELOG.md`。
2. **安装器必须保持零第三方依赖**，除非先改设计并说明取舍。
3. **测试走生产路径。** 安全边界（路径穿越、哈希、原子覆盖）必须有测试。
4. **许可证。** 贡献即同意按本仓库 `LICENSE`（PolyForm Noncommercial License 1.0.0）分发，并保留 `NOTICE` 的署名要求。
