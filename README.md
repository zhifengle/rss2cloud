# rss2cloud

将 RSS 订阅离线下载到 115 网盘。

支持批量添加 magnet, ed2k, http 链接到 115 离线任务

## 关于

基于 [Nahuimi/elevengo](https://github.com/Nahuimi/elevengo)

支持 RSS 源: nyaa, dmhy, mikanni, share.acgnx.net

已添加的 RSS 任务记录保存在本地的同一目录下面的 db.sqlite 文件里

Rust 版本 [rss2pan](https://github.com/zhifengle/rss2pan) 使用的 Web API 添加离线任务。

移除读取浏览器 cookies 的功能。需要此功能使用 [gcookie](https://github.com/zhifengle/gcookie)

```bat
REM 使用 gcookie 读取浏览器的 cookie
gcookie.exe 115.com > .cookies
REM rss2cloud 会读取 .cookies 文件
rss2cloud.exe
```

## 用法

在同一目录下面，配置好 `rss.json` 和 `node-site-config.json`

在命令行运行 `rss2cloud`

```bash
# 查看帮助
rss2cloud -h
# 直接运行。读取 rss.json，依次添加离线任务
rss2cloud
# 使用二维码登录
rss2cloud -q
# 使用cookies
rss2cloud --cookies "yourcookies"

# 指定 rss URL 离线下载
# 如果 rss.json 存在这条url 的配置，会读取配置。没有配置，默认离线到 115 的默认目录
rss2cloud -u "https://mikanani.me/RSS/Bangumi?bangumiId=2739&subgroupid=12"
# --no-cache 跳过检查 db.sqlite 里面缓存的
rss2cloud --no-cache -u "https://mikanani.me/RSS/Bangumi?bangumiId=2739&subgroupid=12"
# --clear-task-type 清除离线任务。 1: 已完成的  2: 所有任务 3: 失败任务 4: 运行的任务 5: 完成并删除的任务 6: 所有的任务
# 清除115任务列表里面已经完成的任务
rss2cloud --clear-task-type 1

# 查看 magnet 子命令帮助
rss2cloud magnet -h
rss2cloud magnet --link "magnet:?xt=urn:btih:12345" --cid "12345" --savepath "文件夹名称"
# 离线包含 magnet 的 txt 文件; 按行分割
rss2cloud magnet --txt magnet.txt --cid "12345" --savepath "文件夹名称"
# savepath 支持多级路径
rss2cloud magnet --link "magnet:?xt=urn:btih:12345" --cid "12345" --savepath "文件夹名称/子文件夹名称"
```

### 文件管理命令

`rss2cloud fs` 提供 115 网盘文件管理命令，可用于搜索整理和目录扁平化。

```bash
# 查看 fs 子命令帮助
rss2cloud fs -h

# 搜索 /anime 下名称包含 episode 的文件，并移动到 /anime/inbox
rss2cloud fs search-mv /anime episode /anime/inbox

# 只移动匹配关键字的 mkv 视频文件
rss2cloud fs search-mv /anime episode /archive --type 4 --ext mkv

# 将 /video_folder 的后代文件提升到目录本身
rss2cloud fs flatten /video_folder

# 只查看 flatten 计划，不执行实际移动
rss2cloud fs flatten /video_folder --dry-run
```

### fs shell

`rss2cloud fs shell` 提供一个交互式文件管理会话，适合连续执行 `ls`、`cd`、`mv`、`search-mv`、`flatten` 这类操作。

```bash
# 启动交互式 shell
rss2cloud fs shell

# 调整目录缓存时间
rss2cloud fs --list-cache-ttl 5s shell

# 关闭目录缓存
rss2cloud fs --list-cache-ttl 0 shell
```

进入 shell 后可用的常见命令：

```text
pwd
ls
cd /anime
stat episode.mkv
mkdir inbox
mv episode.mkv inbox
search-mv /anime episode /archive
flatten /video_folder
rm /anime/old-file.txt
refresh
exit
```

交互示例：

```text
$ rss2cloud fs shell
115:/>
ls
d  123456789012  anime
d  223456789012  archive
cd /anime
search-mv /anime episode /archive
moved: episode-01.mkv
moved: episode-02.mkv
flatten /anime/season1
flattened /anime/season1: moved 12 file(s), removed 3 directory(s)
exit
```

说明：

- shell 需要在交互式终端中运行。
- 支持历史记录。
- 非 Windows 平台支持命令行编辑、历史搜索和路径自动补全。
- Windows 平台当前使用稳定的逐行交互模式，不提供 readline 风格的命令行编辑和自动补全。
- `search_mv` 也可以作为 `search-mv` 的别名使用。
- `refresh` 会清空当前会话缓存，适合在外部有变更后手动刷新。

### 服务模式

```bash
# 查看 server 子命令帮助
rss2cloud server -h
# 运行服务
rss2cloud server
# 添加任务
curl -d '{"tasks": ["magnet:?xt=urn:btih:xx"], "cid":"12345", "savepath":"文件夹名称"}' -X POST http://localhost:8115/add
```

POST `http://localhost:8115/add`

body 示例：

```json
{
  "tasks": ["magnet:?xt=urn:btih:xxx"],
  "cid": "12345",
  "savepath": "文件夹名称/子文件夹名称"
}
```

## 配置

<details>
<summary><code><strong>「 点击查看 配置文件 rss.json 」</strong></code></summary>

```json
{
  "mikanani.me": [
    {
      "name": "test",
      "filter": "/简体|1080p/",
      "url": "https://mikanani.me/RSS/Bangumi?bangumiId=2739&subgroupid=12"
    }
  ],
  "nyaa.si": [
    {
      "name": "VCB-Studio",
      "cid": "2479224057885794455",
      "savepath": "文件夹名称",
      "url": "https://nyaa.si/?page=rss&u=VCB-Studio"
    }
  ],
  "sukebei.nyaa.si": [
    {
      "name": "name",
      "cid": "2479224057885794455",
      "savepath": "文件夹名称",
      "url": "https://sukebei.nyaa.si/?page=rss"
    }
  ],
  "share.dmhy.org": [
    {
      "name": "水星的魔女",
      "filter": "简日双语",
      "cid": "2479224057885794455",
      "savepath": "文件夹名称",
      "url": "https://share.dmhy.org/topics/rss/rss.xml?keyword=%E6%B0%B4%E6%98%9F%E7%9A%84%E9%AD%94%E5%A5%B3&sort_id=2&team_id=0&order=date-desc"
    }
  ]
}
```

</details>

配置了 `filter` 后，标题包含该文字的会被离线。不设置 `filter` 默认离线全部

`/简体|\\d{3-4}[pP]/` 使用斜线包裹的正则规则。注意转义规则

cid 是离线到指定的文件夹 id。

savepath 是可选项，用于指定离线任务的保存路径名称；不设置时保持 115 默认行为。

savepath 支持路径形式，例如 `文件夹名称/文件夹名称`。

获取方法: 浏览器打开 115 的文件，地址栏像 `https://115.com/?cid=2479224057885794455&offset=0&tab=&mode=wangpan`

> 其中 2479224057885794455 就是 cid

<details>
<summary><code><strong>「 点击查看 node-site-config.json 配置 」</strong></code></summary>

配置示例。 设置 【httpsAgent】 表示使用代理连接对应网站。不想使用代理删除对应的配置。

```json
{
  "share.dmhy.org": {
    "httpsAgent": "httpsAgent"
  },
  "nyaa.si": {
    "httpsAgent": "httpsAgent"
  },
  "sukebei.nyaa.si": {
    "httpsAgent": "httpsAgent"
  },
  "mikanime.tv": {
    "headers": {
      "Referer": "https://mikanime.tv/"
    }
  },
  "mikanani.me": {
    "httpsAgent": "httpsAgent"
  }
}
```

</details>

### proxy 配置

设置【httpsAgent】会使用代理。默认使用的地址 `http://127.0.0.1:10809`。

> 【httpsAgent】沿用的 node 版的配置。

需要自定义代理时，在命令行设置 Windows: set HTTPS_PROXY=http://youraddr:port

> Linux: export HTTPS_PROXY=http://youraddr:port

<details>
<summary><code><strong>「 点击查看 批处理脚本 」</strong></code></summary>

```batch
@ECHO off
SETLOCAL
CALL :find_dp0
REM set HTTPS_PROXY=http://youraddr:port
rss2cloud.exe  %*
ENDLOCAL
EXIT /b %errorlevel%
:find_dp0
SET dp0=%~dp0
EXIT /b
```

</details>

把上面的 batch 例子改成自己的代理地址。另存为 rss2cloud.cmd 和 rss2cloud.exe 放在一个目录下面。

在命令行运行 rss2cloud.cmd 就能够使用自己的代理的了。

<details>
<summary><code><strong>「 点击查看 配置 Linux 定时任务 」</strong></code></summary>
假设 rss2cloud 目录在 `$HOME` 下面

新建一个 rss2cloud.sh 的文件

```bash
#!/bin/bash
cd "$(dirname "$0")"
#export HTTPS_PROXY=http://youraddr:port
$HOME/rss2cloud/rss2cloud >> $HOME/rss2cloud/logfile.log 2>&1
```

配置定时任务 `10 8 * * * $HOME/rss2cloud/rss2cloud.sh`

不使用 shell 脚本，定时任务这样写 `10 8 * * * cd $HOME/rss2cloud && ./rss2cloud`

</details>
