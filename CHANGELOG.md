# v0.2.4

## 变动

- RSS 订阅配置项（`rss.json` 与 `config.toml`）新增可选 `exclude` 字段（黑名单），标题命中 `exclude` 的条目不会被离线；可与 `filter`（白名单）同时使用，同样支持 `/正则/` 形式。
- 新增 `rss2cloud task` 子命令，列出 115 当前的离线下载任务，支持 `--limit/-n`、`--status/-s`（all|done|failed|running）和 `--full-name` 参数，输出 NAME、SIZE、STATUS、PERCENT 列。
- 新增 `rss2cloud history` 子命令，列出本地 `db.sqlite` 中已提交的离线记录，支持 `--limit/-n`、`--name`（按订阅配置名过滤）和 `--full-name` 参数，输出 ID、TITLE、FEED、CREATED、HASH 列。
- `rss_items` 表新增 `configName`、`configUrl`、`taskHash` 列，记录每条离线任务的来源订阅与 115 任务哈希；`Store` 初始化时执行幂等的 `migrateSchema`，自动为旧库补齐新列。
- `SaveMagnetItems` 改为同时写入订阅配置名、URL 和 115 返回的任务哈希，`addCloudTasks` 在添加离线任务后落库这些信息。
- `p115.Agent` 新增 `ListOfflineTasks` 方法，按状态过滤并分页拉取 115 离线任务，达到 `limit` 提前停止，拉取不完整时返回错误。
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
