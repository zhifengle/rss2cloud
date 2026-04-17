# v0.2.3

## 变动

- 新增统一 `config.toml` 配置方案，支持认证、服务端口、数据库、115 参数、代理、RSS 订阅和站点配置。
- 支持从当前工作目录和 `~/.config/rss2cloud` 读取配置文件，保留 `rss.json`、`node-site-config.json`、`.cookies` 的兼容读取。
- 新增 `[database].path` 配置，支持通过 TOML 指定 SQLite 数据库路径，并兼容读取既有 `db.sqlite`。
- 优化 cookies 读取和二维码登录写回逻辑，优先遵循已加载配置和 `[auth].cookies_file`。
- 优化 `store.NewWithPath`，创建数据库父目录并向上返回打开数据库和初始化表结构的错误。
- 重写 `scripts/install-release.sh`，支持 `install`、`update`、`uninstall`、`purge`，默认安装 release 二进制并注册 systemd 服务。
- Linux systemd 安装默认使用 `/var/lib/rss2cloud/config.toml`、`/var/lib/rss2cloud/.cookies` 和 `/var/lib/rss2cloud/db.sqlite`

# v0.2.2

## 变动

- 补充说明 `savepath` 支持路径形式，例如 `文件夹名称/文件夹名称`
- 将上游适配到 `Nahuimi/elevengo`，并通过 `go.mod` 的 `replace` 规则兼容其当前模块声明
- 适配新增的离线任务可选参数 `savepath`
- `rss.json` 配置项新增可选字段 `savepath`
- `rss2cloud magnet` 子命令新增 `--savepath` 参数
- 服务模式 `/add` 接口请求体新增可选字段 `savepath`
- 更新 README 与示例配置，补充 `savepath` 的使用说明
