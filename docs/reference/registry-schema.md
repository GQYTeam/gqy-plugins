# registry.json 参考

`schema_version: 1`。未知字段拒绝。所有时间使用 RFC3339。

## 文档

```json
{
  "schema_version": 1,
  "updated_at": "2026-11-02T21:25:00+08:00",
  "plugins": [Plugin]
}
```

## Plugin

```json
{
  "id": "show-images-default",
  "name": "图片展示示例",
  "summary": "一句话简介",
  "description": "可选长描述",
  "homepage": "https://example.com",
  "tags": ["图片", "示例"],
  "versions": [Version]
}
```

约束：

- `id`：`^[a-z0-9][a-z0-9_-]{0,31}$`，全目录唯一。
- `name`、`summary`：非空。
- `homepage`：可选；`http(s)` 绝对地址；strict 模式要求 HTTPS。
- `versions`：至少一项，版本号不重复。

## Version

```json
{
  "version": "1.0.0",
  "channel": "stable",
  "published_at": "2026-11-02T21:25:00+08:00",
  "mode": "resident",
  "protocol": "gqy-jsonrpc-v1",
  "actions": ["list", "show"],
  "provides": [],
  "requires": [],
  "url": "https://github.com/GQYTeam/gqy-plugins/releases/download/v0.1.0/show-images-default-1.0.0.tar.gz",
  "sha256": "cc5b425a3e22cff38a37d5513ec516a916b4156bd30c88ee729ff33dcbf5e156",
  "size": 31375314,
  "yanked": false
}
```

约束：

- `version`：`^\d{1,4}\.\d{1,4}\.\d{1,4}$`。
- `mode`：`resident` / `oneshot`；安装 CLI 只接受 resident。
- `protocol`：`gqy-jsonrpc-v1` / `mcp` / `unix` / `tcp`。
- `url`：非空；strict 模式必须为 HTTPS 绝对地址。
- `sha256`：64 位小写十六进制。
- `size`：非负整数；大于 0 时安装器严格比对下载字节数。
- `yanked`：为 true 时不参与最新版选择，显式安装也拒绝。

## 解析顺序

1. `id` 精确匹配插件。
2. 指定 `version` 时精确匹配，`yanked` 拒绝。
3. 未指定 `version` 时，在非 yanked 版本中按 MAJOR、MINOR、PATCH 数值取最大。
