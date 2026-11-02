## 动机

<!-- 为什么改；关联 issue 用 Closes #123 -->

## 改动

<!-- 一句话说清做了什么；关键实现取舍写在这里 -->

## 影响面

- [ ] 目录条目 / 安装行为有变更（写清 id、version、兼容性）
- [ ] 只动文档、测试或 CI
- [ ] 动了 `docs/` 与 `CHANGELOG.md`

兼容性说明：

## 怎么验的

```bash
# 跑过的命令与结果
```

## 目录条目检查

- [ ] 产物已上传到不可变 HTTPS Release 地址
- [ ] `sha256` 与 `size` 是产物真实值，不是占位值
- [ ] `version` 未被复用，`updated_at` 已更新
- [ ] `go run ./cmd/gqy-plugin verify --registry registry.json --strict --artifacts` 通过

## 基线检查

- [ ] `make check` 或等价命令全绿
- [ ] 没有提交密钥、令牌、构建产物、Release 二进制
- [ ] 文档与代码一致
