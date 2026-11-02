# installer · CLI 与安装蓝图

## 1. 状态

已实现（2026-11-02）。对应设计：REG5、REG6、REG7、REG10。

## 2. 代码落点

| 位置 | 职责 |
| --- | --- |
| `cmd/gqy-plugin/main.go` | 子命令分发与帮助 |
| `cmd/gqy-plugin/catalog.go` | `list / search / info` |
| `cmd/gqy-plugin/install.go` | `install` 参数、目录解析、可选 GQY reload/load |
| `cmd/gqy-plugin/verify.go` | 目录 strict 校验与产物校验 |
| `cmd/gqy-plugin/package.go` | 确定性打包 |
| `internal/manifest` | 包内 `manifest.json` 的 P3 核心校验 |
| `internal/archive` | 确定性 tar.gz 打包、安全解包、插件根定位 |
| `internal/install` | 下载、SHA-256、原子安装、产物校验 |
| `internal/gqyapi` | `POST /api/plugins/reload` 与 `/load` |

## 3. 命令

```bash
# 浏览
gqy-plugin list
gqy-plugin search 图片
gqy-plugin info show-images-default

# 安装
gqy-plugin install show-images-default
gqy-plugin install emote-pack-default@1.0.0 --force
gqy-plugin install show-images-default --dry-run
gqy-plugin install show-images-default --server http://127.0.0.1:5173 --load

# 校验与发布
gqy-plugin verify --strict
gqy-plugin verify --artifacts
gqy-plugin package ./my-plugin --output dist/my-plugin-1.0.0.tar.gz
```

环境变量：

| 变量 | 用途 |
| --- | --- |
| `GQY_PLUGIN_REGISTRY` | 覆盖目录地址 |
| `GQY_HOME` | 安装根，实际目标 `$GQY_HOME/plugins/<id>` |
| `GQY_AUTH_TOKEN` | 管理接口 Bearer token |
| `GQY_URL` | `--server` 的默认值 |

## 4. 安装顺序

1. `registry.Load` → `Find(id)` → `ResolveVersion(version)`。
2. 拒绝 `mode:"oneshot"`。
3. 创建 `$GQY_HOME/plugins`，持有 `<plugins>/.install-<id>.lock` 目录锁。
4. 在 plugins 目录下建 `.gqy-plugin-stage-<id>-*`，下载 `plugin.tar.gz`。
5. 边下边算 SHA-256；对比目录 `sha256`，有 `size` 时对比字节数。
6. 解包到 stage/unpacked；拒绝路径穿越、绝对路径、符号链接与设备文件；文件数上限 10000、解包后上限 512 MiB。
7. 找到包根 `manifest.json`；校验 id、version、mode、protocol 与目录一致。
8. 目标已存在且无 `--force`：报错。`--force` 时先 rename 到同目录备份。
9. rename 插件根到 `plugins/<id>`；失败恢复备份；成功后删除备份与 stage。
10. `--dry-run` 在步骤 7 后返回，不落盘。
11. 安装完成后才处理可选的 `--server` / `--load`。

## 5. 打包格式

- `.tar.gz`，包根直接包含 `manifest.json`；也接受单一顶层目录包。
- 打包器输出确定性归档：路径排序、固定 uid/gid、固定时间戳、权限归一为 `0644` / `0755`。
- 排除 `.git`、`node_modules`、`.gqy`、`dist`、`coverage`、`.DS_Store`、`*.log`、`.env`（保留 `.env.example`）。
- 输出摘要 JSON：

```json
{
  "path": "dist/demo-1.0.0.tar.gz",
  "id": "demo",
  "version": "1.0.0",
  "size": 12345,
  "sha256": "64 位小写十六进制"
}
```

## 6. 输出与退出码

- 成功：退出码 0；`--json` 输出结构化结果。
- 参数、网络、哈希、manifest 或落盘失败：退出码 1，错误写 stderr。
- `--server` 通知失败不推翻已经成功的安装，作为 warning 输出；安装事实不回滚。
- `--dry-run` 结果中 `dry_run: true`，目录不改变。
- 日志与错误不输出 token；GQY 管理接口使用 `Authorization: Bearer`，token 只来自 flag 或环境变量。
