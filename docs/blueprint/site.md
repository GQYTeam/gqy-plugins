# site · 静态目录站蓝图

## 1. 状态

已实现（2026-11-02）。对应设计：REG1、REG8、REG10。

## 2. 文件与视觉

```text
web/
  index.html   # 深色 skill 风格目录、终端式快速命令、安装说明
  styles.css   # 原生 CSS；黑色背景、等宽标签、卡片网格
  app.js       # fetch('./registry.json')、搜索、来源标识、复制安装命令
```

视觉对标 agent skill 目录：大字号 hero、深色终端块、加粗标题、等宽元信息、紧凑卡片。没有构建步骤、外部字体、第三方脚本或后端。

## 3. 数据

- 只请求同目录的 `registry.json`（Pages 发布目录）。
- 卡片取每个插件最新非 yanked 版本。
- 搜索匹配 `id / name / summary / description / tags`，全部转为小写；`/` 聚焦搜索框。
- `source_type=org` 显示 `ORG · AUTO`，`external` 显示 `EXTERNAL · PR`。
- `mode=oneshot` 显示 `ONESHOT`，卡片不提供常驻安装按钮。

## 4. 安装命令

**所有安装命令第一步都是安装 CLI，第二步才是安装插件并自动加载：**

```bash
go install github.com/GQYTeam/gqy-plugins/cmd/gqy-plugin@latest
gqy-plugin install <id>@<version> --server http://127.0.0.1:5173 --load
```

- 卡片按钮“安装并加载”复制包含 `&&` 的完整两段命令；不会在浏览器执行任何本地程序。
- `--server` 调用 `/api/plugins/reload`，`--load` 请求 `/api/plugins/<id>/load`。
- 卡片同时提供可展开的完整命令原文，便于检查后再执行。

## 5. 页面

| 区块 | 内容 |
| --- | --- |
| Hero | `GQY PLUGINS` 大字、目录定位、CLI 优先的终端命令、复制按钮 |
| 工具栏 | 搜索、当前可见数量、更新时间 |
| 卡片 | 图标、来源、版本、一次性标记、简介、标签、动作、协议、大小、安装并加载按钮、命令详情 |
| 安装说明 | 明确“先装 CLI → 再装插件 → 自动 reload/load” |
| 页脚 | 信任边界与安全说明 |

## 6. 发布

`Publish catalog` workflow：

触发：`main` push、每 6 小时、手动 dispatch。

1. 检出 `main`。
2. 运行组织内插件同步，生成 `registry.json`。
3. strict 校验。
4. 组装 `_site/`（`web/` + 生成目录），上传 Pages artifact。
5. 把生成目录 fast-forward 到 `published` 分支。
6. `actions/deploy-pages` 部署。

页面地址：`https://gqyteam.github.io/gqy-plugins/`。
线上目录：`https://raw.githubusercontent.com/GQYTeam/gqy-plugins/published/registry.json`。

## 7. 故障显示

| 情况 | 页面行为 |
| --- | --- |
| 请求失败或 HTTP 非 200 | 显示“读取目录失败”与原因 |
| `plugins` 不是数组 | 显示 schema 错误 |
| 搜索无结果 | 显示空结果提示，不隐藏搜索框 |
| 插件无可用版本 | 卡片显示“暂无可用版本”，按钮禁用 |
| 插件为 oneshot | 卡片显示“一次性插件”，不提供常驻安装 |
