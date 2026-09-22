# v0.2.5

## 变动

- 修复未配置 `exclude` 时 RSS 条目被全部过滤、导致显示 `0 task` 的问题。
- 增加 Mikanani 搜索 RSS 样本解析测试，校验条目标题、Episode 链接和磁力链提取。
- 增加 `filter`、`exclude` 及正则筛选的回归测试。
- 移除依赖 `mockey` 的测试与模块依赖；现在可直接运行 `go test ./...`，无需额外编译参数。
- 默认跳过 Dmhy 和 Acgnx 的在线 RSS 探测测试；设置 `RSS2CLOUD_TEST_LIVE=1` 后可手动执行。
