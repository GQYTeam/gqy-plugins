# v0.2.0 · 来源分层自动收录

## 重点

- **组织内自动收录**：`gqy-plugin sync-org` 扫描 `GQYTeam/gqy-plugins-*` 公开仓库，每 6 小时由 workflow 自动更新 `registry.json`。
- **组织外自动 PR**：外部 GitHub 仓库通过 issue 或手动 dispatch 提交，自动校验 Release 产物并创建 PR；外部路径不会直接写 main。
- **来源字段**：目录增加 `source_type`（`org` / `external`）与 `repository`，站点与审计可以区分来源。
- **main ruleset**：保留 PR 评审与两个 Go status check；仅管理员与同步 bot 可 bypass。
- 新增组织内插件 `github-account-default@1.0.0` 与 `migrator-default@1.0.0`（commit-pinned tarball；后者为 oneshot）。
- 线上目录改为 `published` 分支自动生成；main 保留可审计基目录，CLI 默认读取发布目录。
- 网站改为 skill 风格卡片；安装命令先 `go install` CLI，再安装插件并 `--load` 自动加载。

## 安装

```bash
go run github.com/GQYTeam/gqy-plugins/cmd/gqy-plugin@latest list
```

已有 `show-images-default@1.0.0` 与 `emote-pack-default@1.0.0` 的 Release 产物和哈希不变。

## 验证

- `go test -race ./...`
- `gqy-plugin verify --registry registry.json --strict --artifacts`
- 组织同步重复运行不产生目录变化
- 外部收录 workflow 通过 actionlint
