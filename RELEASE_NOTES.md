# v0.1.0 · 首批官方插件目录

这是 `gqy-plugins` 在线目录与安装器的首个可用版本。

## 收录插件

- `show-images-default@1.0.0`：13 张示例图片，`list` / `show`
- `emote-pack-default@1.0.0`：24 张表情 + 4 个动图，`list` / `show` / `match`

## 安装

```bash
go run github.com/GQYTeam/gqy-plugins/cmd/gqy-plugin@latest install show-images-default
```

安装器会下载 Release 资产、校验 `registry.json` 中登记的 SHA-256 与大小、拒绝路径穿越，最后原子写入 `$GQY_HOME/plugins/<id>/`。默认不加载插件。

## 完整性

- `registry.json` 由 CI 做 strict 校验。
- Release 产物与 `checksums.txt` 一同发布。
- `gqy-plugin verify --strict --artifacts` 可逐一下载并核对哈希、大小与包内 manifest。

## 信任边界

安装插件 = 信任它。SHA-256 只保证传输完整，不提供代码签名或沙箱。只安装你信任来源的插件。
