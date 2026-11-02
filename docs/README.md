# 文档索引

`gqy-plugins` 的文档入口。仓库还在骨架阶段，这里只列**当前真实存在**的文档；设计稿、蓝图、施工记录在真正开写之前不建空目录、不留死链。

## 仓库内文档

| 文档 | 内容 |
| --- | --- |
| [`../README.md`](../README.md) | 仓库入口：`registry.json` schema、校验规则、验收命令、新增条目流程、路线图、许可与署名 |
| [`../CONTRIBUTING.md`](../CONTRIBUTING.md) | 分支 / PR / 提交规范与本地验收（`make check`） |
| [`../SECURITY.md`](../SECURITY.md) | 私密漏洞披露流程与支持范围 |
| [`../CHANGELOG.md`](../CHANGELOG.md) | Keep a Changelog 风格的变更记录 |
| [`../NOTICE`](../NOTICE) | 来源与署名要求（含四项署名格式） |
| [`../registry.json`](../registry.json) | 官方插件目录（唯一事实来源） |

## 代码即文档

- 目录字段的真实定义在 [`../internal/registry/registry.go`](../internal/registry/registry.go) 的 `Document` / `Plugin` / `Version`；README 的字段表与它保持一致，不一致时以代码为准，并顺手更正 README。
- 校验规则同样以 `registry.go` 为准（`Parse`、`Validate`、`validatePlugin`、`validateVersion`、`validateArtifactURL`）。
- 目录行为由 [`../internal/registry/registry_test.go`](../internal/registry/registry_test.go) 覆盖：仓库根 `registry.json` 必须能解析通过，未知字段 / 错误 `schema_version` / 非法 id / 非法 sha256 / 非法版本号 / strict HTTPS 必须被拒绝。

## 尚未建立的部分

- `docs/designs/`、`docs/blueprint/`、`docs/construction/`：设计、蓝图与施工记录。等工作真正开始再建目录，不预置空文档。
- `docs/third-party/`：第三方代码、素材、参考项目的许可与出处。本仓库目前只依赖 Go 标准库，暂无第三方内容；一旦引入，按组织标准登记在这里，并在 README 的「许可与署名」小节指向它。
