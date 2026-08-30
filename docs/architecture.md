# rss2cloud 架构文档

## 1. 项目定位

`rss2cloud` 是一个将 RSS 订阅条目转换为 115 网盘离线任务的 Go 命令行工具，同时提供一个简易 HTTP 服务接口用于批量提交磁力链接。

项目当前包含两条并行演进的能力线：

1. 从 RSS 订阅源提取磁力链接并提交到 115。
2. 接收外部传入的 magnet/ed2k/http 链接并提交到 115。
3. 提供面向 115 的云盘文件系统抽象、一次性 `fs` 命令和交互式 shell。

## 2. 当前对外输出

项目运行后的主要输出不是“生成文件”，而是以下几类结果：

### 2.1 业务输出

- 向 115 提交离线下载任务。
- 按配置中的 `cid` 将任务投递到指定目录。
- 通过本地 SQLite 缓存跳过已处理过的 magnet。

### 2.2 本地文件输出

- `db.sqlite`
  - 本地缓存数据库。
  - 记录已经处理过的 RSS 条目和 magnet，用于避免重复投递。
- `.cookies`
  - 115 登录凭证缓存。
  - 启用二维码登录后，成功登录会落盘，后续优先复用。
- `qrcode115.png`
  - 二维码登录时的临时文件。
  - 登录成功后会删除；异常退出时可能残留。

### 2.3 终端 / 服务输出

- CLI 模式下输出日志，例如：
  - 配置未找到
  - RSS 抓取失败
  - 某个订阅新增了多少任务
  - 有多少条目没有提取到 magnet
- Server 模式下输出：
  - 服务启动端口
  - 收到退出信号后的优雅关闭信息
- `fs` 一次性命令输出：
  - 当前目录
  - 目录列表
  - 单对象元信息
  - `mv/cp/flatten/search-mv` 等命令的执行摘要或 JSON
- `fs shell` 输出：
  - `provider:pwd> ` prompt
  - 目录列表与对象元信息
  - shell 帮助、错误和复合命令执行摘要

### 2.4 额外本地状态文件

- `.rss2cloud_shell_state.json`
  - `fs shell` 会话状态。
  - 当前用于保存上一次 `cwd`。
- `.rss2cloud_shell_history`
  - `fs shell` 历史记录文件。
  - 当前使用 `reeflective/readline` 的 JSONL 历史格式。

## 3. 运行模式

当前架构可以分成四种运行入口：

### 3.1 CLI 默认模式

执行 `rss2cloud` 后：

1. 初始化 115 Agent。
2. 读取 `rss.json`。
3. 遍历所有 RSS 配置。
4. 拉取订阅、过滤条目、提取 magnet。
5. 查重后批量提交到 115。
6. 将已提交条目写入 `db.sqlite`。

### 3.2 CLI 单次任务模式

- `rss2cloud -u <rss-url>`
  - 仅处理单个 RSS URL。
- `rss2cloud magnet --link ...`
  - 直接提交单个 magnet/ed2k/http 链接。
- `rss2cloud magnet --text ...`
  - 从文本文件逐行读取链接后批量提交。
- `rss2cloud --clear-task-type <n>`
  - 调用 115 清理离线任务。

### 3.3 HTTP Server 模式

执行 `rss2cloud server` 后：

1. 初始化 115 Agent。
2. 启动 HTTP 服务，默认监听 `:8115`。
3. 暴露 `POST /add` 接口。
4. 将请求中的任务列表直接转交给 `Agent.AddMagnetTask()`。

这一路径不会经过 RSS 抓取模块，但仍共享同一个 115 Agent 和本地数据库句柄。

### 3.4 `fs` 文件系统模式

执行 `rss2cloud fs ...` 后：

1. 初始化 115 Agent。
2. 构造 `p115.FileSystem` provider。
3. 初始化 `cloudfs.Session`。
4. 执行一次性命令或进入交互式 shell。

当前已经存在的入口包括：

- 一次性命令：
  - `pwd`
  - `ls`
  - `stat`
  - `mkdir`
  - `rename`
  - `mv`
  - `cp`
  - `rm`
  - `flatten`
  - `search-mv`
- 交互式 shell：
  - `rss2cloud fs shell`

## 4. 模块划分

当前代码可以理解为 7 个层次清晰但耦合度不低的模块。

### 4.1 入口层

- `main.go`
  - 程序入口，仅调用 `cmd.Execute()`。
- `cmd/rss2cloud.go`
  - CLI 编排层。
  - 负责解析命令行参数、初始化 115 Agent、切换运行模式。

### 4.2 业务编排层

- `p115/p115.go`
  - 当前项目的核心服务层。
  - 职责包括：
    - 115 登录初始化
    - cookies 读取与保存
    - RSS 任务执行
    - 本地缓存查重
    - 分块提交离线任务
    - 清理 115 离线任务

`p115.Agent` 是当前系统的中心对象，连接了 115 API、RSS 解析结果和本地存储。

### 4.3 文件系统抽象层

- `cloudfs/*.go`
  - 提供云盘文件系统抽象。
  - 当前包括：
    - `Entry` / 错误模型
    - 组合接口与聚合 `Driver`
    - `Session`
    - 目录列表缓存与 `Refresh`
    - `SearchMove`
    - `Flatten`
    - `CooldownLimiter` / `RateLimitedDriver`

- `p115/filesystem.go`
  - 115 provider 适配层。
  - 负责把 `elevengo.Agent` 包装为 `cloudfs` 能力接口。

### 4.4 RSS 解析层

- `rsssite/rsssite.go`
  - RSS 通用流程：
    - 拉取 feed
    - 应用过滤规则
    - 将 feed item 转换为 `MagnetItem`
- `rsssite/rssconfig.go`
  - 读取 `rss.json`
  - 按 URL 查找配置
- `rsssite/*.go`
  - 面向不同站点的适配器：
    - `nyaa.go`
    - `mikanani.go`
    - `dmhy.go`
    - `acgnx.go`
    - `rsshub.go`

这里采用了一个很轻量的“站点适配器”模式：不同网站实现统一的 `MagnetSite` 接口，从各自 RSS 条目的不同字段中提取 magnet。

### 4.5 请求层

- `request/request.go`
  - HTTP 请求封装。
  - 支持：
    - 站点级请求头注入
    - 代理配置
    - gzip/deflate 解压
    - 按 host 复用 `http.Client`
    - Cookie Jar 维持会话

该层为 RSS 拉取和部分未来站点交互提供基础能力。

### 4.6 存储层

- `store/store.go`
  - 使用 SQLite 持久化本地状态。
  - 当前主要能力：
    - 初始化表结构
    - 保存已处理 `MagnetItem`
    - 基于 magnet 精确查重
    - 基于 `xt` 模糊查重（预留替代方案）

### 4.7 服务接口层

- `server/server.go`
  - 负责启动 HTTP 服务。
  - 当前接口很薄，只是把外部请求转发给 `p115.Agent`。

## 5. 核心数据流

### 5.1 RSS 到 115 的主流程

```text
CLI / 定时任务
  -> cmd.Execute
  -> p115.initAgent
  -> p115.Agent.ExecuteAllRssTask
  -> rsssite.ReadRssConfigDict
  -> rsssite.GetMagnetItemList
  -> request.Get / gofeed.ParseString
  -> 站点适配器提取 magnet
  -> store.HasItem 查重
  -> elevengo.Agent.OfflineAddUrl 提交到 115
  -> store.SaveMagnetItems 持久化
```

### 5.2 HTTP 接口提交流程

```text
HTTP POST /add
  -> server.handleAddTask
  -> p115.Agent.AddMagnetTask
  -> elevengo.Agent.OfflineAddUrl
```

### 5.3 登录流程

```text
启动
  -> 读取命令行 cookies / .cookies / 二维码登录
  -> p115.NewAgent / p115.QrcodeLogin
  -> elevengo.CredentialImport 或 QrcodeStart + QrcodePoll
  -> 成功后缓存到 .cookies
```

### 5.4 `fs` 文件操作流程

```text
CLI / shell
  -> cmd/fs*.go
  -> initFsSession
  -> p115.Agent.FileSystemWithOption
  -> cloudfs.NewSession
  -> Session.Resolve / Ls / Mkdir / Rename / Mv / Cp / Rm / SearchMove / Flatten
  -> p115.FileSystem
  -> elevengo.Agent.FileGet / FileIterate / DirMake / FileRename / FileMove / FileCopy / FileDelete / FileSearch
```

## 6. 关键配置与状态

### 6.1 `rss.json`

职责：定义每个 RSS 订阅的抓取规则。

关键字段：

- `name`
  - 任务名称，仅用于日志标识。
- `url`
  - RSS 地址。
- `cid`
  - 115 目标目录 id。
- `filter`
  - 标题过滤规则（白名单）。
  - 支持普通子串匹配，也支持 `/.../` 包裹的正则。
- `exclude`
  - 标题排除规则（黑名单）。匹配的标题不会被离线。
  - 支持普通子串匹配，也支持 `/.../` 包裹的正则。

### 6.2 `node-site-config.json`

职责：定义站点级请求行为。

当前主要支持：

- `httpsAgent`
  - 打开代理。
- `headers`
  - 注入自定义请求头，如 `Referer` 或 `Cookie`。

### 6.3 `db.sqlite`

当前包含两张表：

- `rss_items`
  - 已处理 RSS 条目缓存。
- `sites_status`
  - 已建表，但当前没有实际使用路径。

### 6.4 `fs` 相关配置与状态

- `--cwd`
  - 设置一次性命令或 shell 启动时的起始目录。
- `--root-id`
  - 覆盖 provider 逻辑根。
- `--page-size`
  - 当前为保留配置，初始化链路已接入，但尚未改变实际列举行为。
- `--op-rate-limit-min-ms` / `--op-rate-limit-max-ms`
  - 操作层 limiter 配置。
- `--list-cache-ttl`
  - `Session` 目录列表缓存 TTL。

## 7. 当前架构特点

### 7.1 优点

- 入口简单，CLI 和 Server 两种模式共享同一套核心能力。
- RSS 站点适配器结构足够轻，扩展新站点成本不高。
- 请求层抽象独立，代理和头部策略集中管理。
- 本地 SQLite 缓存使“重复提交同一个 magnet”这个核心问题有了稳定兜底。

### 7.2 当前的耦合点

- `p115.Agent` 同时承担了登录、流程编排、缓存查重、任务提交等多种职责，是典型的中心化服务对象。
- `request` 与 `rsssite` 使用包级全局配置，运行期动态切换配置的能力较弱。
- `store` 直接在初始化时创建表结构，数据库生命周期和业务对象生命周期绑定较紧。

## 8. 当前已知问题与风险

以下内容基于当前代码实现，而不是未来目标设计。

### 8.1 测试可运行性不稳定

- `p115` 测试依赖 `mockey`，当前环境直接执行 `go test ./...` 会失败，需要额外 `gcflags` 或关闭检查。
- `store` 测试依赖 `go-sqlite3`，在 `CGO_ENABLED=0` 环境下会失败。
- `rsssite` 中仍有依赖真实公网 RSS 的测试，容易因为网络波动失败。

这说明当前测试分层尚未完全收敛到“默认可离线执行”的状态。

### 8.2 持久化时机和粒度需要进一步校验

- `p115.addCloudTasks()` 在分块提交成功后会调用 `SaveMagnetItems`。
- 但当前保存调用位于分块循环内部，且传入的是整个 `filterdItems`，不是当前分块 `items`。

从设计意图看，落库粒度更合理的方式应当是“当前成功提交的分块”。否则在多分块场景下，缓存写入可能与真实成功提交集合不完全一致。

### 8.3 配置热更新能力缺失

- `request.ReadNodeSiteConfig()` 的结果在包初始化阶段赋值给 `ReqSiteConfig`。
- 程序启动后若配置文件变化，默认不会自动重载。

### 8.4 数据模型仍有未完成设计

- `RssConfig.Expiration` 已定义但未参与调度或过滤。
- `sites_status` 表已创建但没有写入或查询逻辑。
- `HasMagnetByXt()` 已标注为替代方案，但尚未进入主流程。

### 8.5 `fs/cloudfs` 线已进入收口阶段

- 一次性 `fs` 命令、`fs shell`、`flatten`、`search-mv` 都已落地。
- 真实 115 账号验证结果尚未沉淀成正式验证报告。
- `page-size` 仍是保留位，没有真正控制底层分页行为。
- shell 与一次性命令的输出风格已经部分收口，但还不是最终统一策略。

## 9. 演进建议

如果后续要继续演进，这个项目比较自然的方向有四条：

1. 把 `p115.Agent` 拆成“认证层、任务提交层、RSS 编排层”。
2. 把缓存写入从“整批保存”收敛到“按成功分块保存”。
3. 把站点适配器和请求配置做成更显式的注册机制，减少包级全局状态。
4. 把 `fs/cloudfs` 线从“功能已落地”推进到“真实验证完成、输出与文档收口”。

## 10. 一句话总结

当前 `rss2cloud` 的架构可以概括为：

一个以 `p115.Agent` 为中心的命令行/服务双入口应用，前面接 RSS 站点适配器与 HTTP 请求层，后面接 115 API 与 SQLite 本地缓存，用最小结构完成“抓取 RSS -> 提取 magnet -> 去重 -> 提交 115”这条主业务链路。
