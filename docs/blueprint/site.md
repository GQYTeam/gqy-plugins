# site · 静态目录站蓝图

## 1. 状态

已实现（2026-11-02）。对应设计：REG1、REG8、REG10。

## 2. 文件

```text
web/
  index.html   # 语义结构与安装说明
  styles.css   # 原生 CSS，亮/暗色跟随系统
  app.js       # fetch('./registry.json')、搜索、复制命令
```

站点没有构建步骤、没有外部字体与脚本、没有后端。

## 3. 数据

- 只请求同目录的 `registry.json`。
- 卡片取每个插件最新非 yanked 版本。
- 搜索匹配 `id / name / summary / description / tags`，全部转为小写。
- 安装命令固定为：

```text
go run github.com/GQYTeam/gqy-plugins/cmd/gqy-plugin@latest install <id>@<version>
```

## 4. 页面

| 区块 | 内容 |
| --- | --- |
| Hero | 目录定位、安全说明、一键复制通用命令 |
| 工具栏 | 搜索框与可见数量 |
| 卡片 | 名称、版本、简介、标签、动作、协议、大小、复制命令 |
| 安装之后 | reload 与 load 两步说明，强调安装不等于执行 |
| 页脚 | 更新时间和安全链接 |

## 5. 发布

GitHub Pages 工作流只在 `main` 分支触发：

1. 检出仓库。
2. 创建 `_site/`，复制 `web/` 内容与根 `registry.json`。
3. `actions/upload-pages-artifact` 上传。
4. `actions/deploy-pages` 部署。

页面地址：`https://gqyteam.github.io/gqy-plugins/`。

## 6. 故障显示

| 情况 | 页面行为 |
| --- | --- |
| 请求失败或 HTTP 非 200 | 显示“读取目录失败”与原因 |
| `plugins` 不是数组 | 显示 schema 错误 |
| 搜索无结果 | 显示空结果提示，不隐藏搜索框 |
| 插件无可用版本 | 卡片显示“无版本”，命令不可用但仍可搜索 |
