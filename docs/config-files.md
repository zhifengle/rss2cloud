# 配置文件说明

`rss2cloud` 支持统一的 TOML 配置文件和传统的 JSON 配置文件。新用户推荐使用 `config.toml` 统一配置，现有用户可以继续使用原有配置文件，无需修改。

## 统一 TOML 配置（推荐）

### 配置文件位置

`config.toml` 查找顺序：

1. 当前工作目录的 `config.toml`
2. `~/.config/rss2cloud/config.toml`

### 完整配置示例

```toml
# 认证配置
[auth]
# 方式 1: 指定 cookies 文件路径（相对路径基于 config.toml 所在目录）
cookies_file = ".cookies"

# 方式 2: 直接配置 cookies 字符串（不推荐，建议使用 cookies_file）
# cookies = "UID=xxx; CID=xxx; SEID=xxx; KID=xxx"

# 服务器配置
[server]
port = 8115  # HTTP 服务器端口，默认 8115

# 数据库配置
[database]
path = "db.sqlite"  # SQLite 数据库文件路径，默认 db.sqlite

# 115 云存储配置
[p115]
disable_cache = false    # 是否禁用缓存，默认 false
chunk_delay = 2          # 分块延迟（秒），默认 2
chunk_size = 200         # 分块大小，默认 200
cooldown_min_ms = 1000   # 冷却最小时间（毫秒），默认 1000
cooldown_max_ms = 1100   # 冷却最大时间（毫秒），默认 1100

# 代理配置
[proxy]
http = "http://127.0.0.1:10809"  # HTTP 代理地址，默认 http://127.0.0.1:10809

# RSS 订阅配置（可配置多个）
[[rss]]
site = "mikanani.me"
name = "测试订阅"
filter = "简体内嵌"
url = "https://mikanani.me/RSS/Bangumi?bangumiId=2739&subgroupid=12"

[[rss]]
site = "share.dmhy.org"
name = "水星的魔女"
savepath = "文件夹名称"
filter = "简日双语"
url = "https://share.dmhy.org/topics/rss/rss.xml?keyword=..."
cid = "123456"           # 可选：115 目录 ID
expiration = 3600        # 可选：缓存过期时间（秒）

# 站点特定配置
[sites."share.dmhy.org"]
https_agent = true  # 是否使用代理

[sites."nyaa.si"]
https_agent = true

[sites."115.com".headers]
Cookie = "UID=xxx; CID=xxx; SEID=xxx; KID=xxx"
```

### 配置字段说明

#### [auth] 认证配置

- `cookies_file`: cookies 文件路径，支持相对路径（相对于 config.toml 所在目录）和绝对路径
- `cookies`: 直接配置 cookies 字符串（不推荐，建议使用 cookies_file）

#### [server] 服务器配置

- `port`: HTTP 服务器端口，有效范围 1-65535，默认 8115

#### [database] 数据库配置

- `path`: SQLite 数据库文件路径，支持相对路径（相对于 config.toml 所在目录）和绝对路径，默认 db.sqlite

#### [p115] 115 云存储配置

- `disable_cache`: 是否禁用缓存，默认 false
- `chunk_delay`: 分块延迟（秒），默认 2
- `chunk_size`: 分块大小，默认 200
- `cooldown_min_ms`: 冷却最小时间（毫秒），默认 1000
- `cooldown_max_ms`: 冷却最大时间（毫秒），默认 1100

#### [proxy] 代理配置

- `http`: HTTP 代理地址，默认 http://127.0.0.1:10809

#### [[rss]] RSS 订阅配置

每个 `[[rss]]` 块定义一个订阅，必填字段：

- `site`: 站点域名（用于分组）
- `name`: 订阅名称
- `url`: RSS 订阅地址

可选字段：

- `cid`: 115 云存储目录 ID
- `savepath`: 保存路径
- `filter`: 内容过滤规则
- `expiration`: 缓存过期时间（秒）

#### [sites."<域名>"] 站点配置

- `https_agent`: 是否为该站点启用代理
- `headers`: 自定义 HTTP 请求头

### 配置优先级规则

配置值按以下优先级合并（从高到低）：

1. **命令行参数**：`--cookies`、`--rss`、`--port` 等
2. **TOML 配置**：`config.toml` 中的配置
3. **传统配置文件**：`rss.json`、`node-site-config.json`、`.cookies`
4. **程序默认值**：内置默认配置

示例：如果同时存在 `config.toml` 中的 `[server].port = 8115` 和命令行参数 `--port 9000`，则使用命令行参数的值 9000。

### 相对路径解析

`[auth].cookies_file` 和 `[database].path` 中的相对路径基于 `config.toml` 所在目录解析：

```toml
# 如果 config.toml 位于 ~/.config/rss2cloud/config.toml
[auth]
cookies_file = ".cookies"  # 解析为 ~/.config/rss2cloud/.cookies

[database]
path = "data/app.db"  # 解析为 ~/.config/rss2cloud/data/app.db
```

## 传统配置文件（向后兼容）

现有用户可以继续使用传统配置文件，无需迁移到 `config.toml`。

### 默认查找顺序

`rss.json`：

1. 当前工作目录的 `rss.json`
2. `~/.config/rss2cloud/rss.json`

`.cookies`：

1. 当前工作目录的 `.cookies`
2. `~/.config/rss2cloud/.cookies`

`node-site-config.json`：

1. 当前工作目录的 `node-site-config.json`
2. `~/.config/rss2cloud/node-site-config.json`
3. `~/node-site-config.json`

`~/node-site-config.json` 是旧行为的兼容路径。

### 显式指定 RSS 配置

命令行参数 `--rss` / `-r` 保持原有语义：传入什么路径就读取什么路径，不走默认查找顺序。

```bash
rss2cloud --rss /path/to/rss.json
```

### 示例目录

```text
~/.config/rss2cloud/
  .cookies
  rss.json
  node-site-config.json
```

### 数据库文件

`db.sqlite` 默认优先读取当前工作目录的既有文件；如果当前目录不存在，会继续查找 `~/.config/rss2cloud/db.sqlite`。两处都不存在时，会在当前工作目录创建新的 `db.sqlite`。

如果使用 TOML 配置，可以通过 `[database].path` 指定数据库文件路径。相对路径按 `config.toml` 所在目录解析。

## 配置迁移指南

从传统配置文件迁移到 `config.toml` 可以逐步进行：

### 步骤 1：创建基础 config.toml

创建 `~/.config/rss2cloud/config.toml`，先配置认证部分：

```toml
[auth]
cookies_file = ".cookies"
```

此时 RSS 和站点配置仍从 `rss.json` 和 `node-site-config.json` 读取。

### 步骤 2：迁移 RSS 订阅

将 `rss.json` 中的订阅添加到 `config.toml`：

```toml
[[rss]]
site = "mikanani.me"
name = "订阅名称"
url = "https://..."
```

添加 `[[rss]]` 配置后，`rss.json` 将不再被读取。

### 步骤 3：迁移站点配置

将 `node-site-config.json` 中的站点配置添加到 `config.toml`：

```toml
[sites."share.dmhy.org"]
https_agent = true
```

添加 `[sites]` 配置后，`node-site-config.json` 将不再被读取。

### 步骤 4：清理（可选）

完成迁移后，可以删除不再使用的传统配置文件：

- `rss.json`
- `node-site-config.json`

保留 `.cookies` 文件，或将其内容直接配置到 `config.toml` 的 `[auth].cookies` 字段（不推荐）。

## 命令行参数

所有命令行参数的优先级最高，会覆盖配置文件中的设置：

- `--cookies`: 指定 cookies 字符串
- `--rss` / `-r`: 指定 RSS 配置文件路径
- `--port`: 指定服务器端口
- `--disable-cache`: 禁用 115 缓存
- `--chunk-delay`: 设置分块延迟
- `--chunk-size`: 设置分块大小
- `--cooldown-min-ms`: 设置冷却最小时间
- `--cooldown-max-ms`: 设置冷却最大时间

示例：

```bash
# 使用命令行参数覆盖配置文件
rss2cloud --port 9000 --cookies "UID=xxx; CID=xxx; SEID=xxx; KID=xxx"
```
