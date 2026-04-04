# v0.2.2

## 变动

- 补充说明 `savepath` 支持路径形式，例如 `文件夹名称/文件夹名称`
- 将上游适配到 `Nahuimi/elevengo`，并通过 `go.mod` 的 `replace` 规则兼容其当前模块声明
- 适配新增的离线任务可选参数 `savepath`
- `rss.json` 配置项新增可选字段 `savepath`
- `rss2cloud magnet` 子命令新增 `--savepath` 参数
- 服务模式 `/add` 接口请求体新增可选字段 `savepath`
- 更新 README 与示例配置，补充 `savepath` 的使用说明
