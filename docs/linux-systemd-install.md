# Linux systemd 安装

本文说明如何在 Linux systemd 环境下安装 `rss2cloud` 并注册为系统服务。

## 安装

```bash
curl -fsSL https://raw.githubusercontent.com/zhifengle/rss2cloud/main/scripts/install-release.sh | sudo bash
```

安装脚本会执行以下操作：

- 下载 GitHub latest release 中的 `rss2cloud-<version>-linux-amd64-musl.tar.gz`。
- 安装二进制到 `/usr/local/bin/rss2cloud`。
- 创建运行目录 `/var/lib/rss2cloud`。
- 按推荐方式初始化 `config.toml` 到 `/var/lib/rss2cloud/config.toml`。
- 创建 cookies 文件 `/var/lib/rss2cloud/.cookies`。
- 写入 systemd service：`/etc/systemd/system/rss2cloud.service`。

当前 release workflow 只发布 `linux amd64` musl 二进制，所以脚本也只支持 Linux amd64。

重复执行安装命令会更新二进制和 systemd service，不会覆盖已有配置、cookies 和数据库。

## 推荐配置目录

服务的 `WorkingDirectory` 是 `/var/lib/rss2cloud`。

因此程序会优先读取当前工作目录的配置：

```text
/var/lib/rss2cloud/config.toml
```

默认生成的配置：

```toml
[auth]
cookies_file = ".cookies"

[server]
port = 8115

[database]
path = "db.sqlite"
```

对应文件位置：

```text
/var/lib/rss2cloud/config.toml
/var/lib/rss2cloud/.cookies
/var/lib/rss2cloud/db.sqlite
```

如果需要继续使用传统配置，也可以把文件放到同一目录：

```text
/var/lib/rss2cloud/rss.json
/var/lib/rss2cloud/node-site-config.json
```

如果手动写入 cookies，可以直接覆盖安装脚本创建的文件：

```bash
sudo install -m 600 /path/to/.cookies /var/lib/rss2cloud/.cookies
```

## 服务管理

配置文件准备好后启动服务：

```bash
sudo systemctl enable --now rss2cloud
```

查看状态：

```bash
sudo systemctl status rss2cloud
```

查看日志：

```bash
sudo journalctl -u rss2cloud -f
```

重启服务：

```bash
sudo systemctl restart rss2cloud
```

服务默认执行：

```bash
/usr/local/bin/rss2cloud server
```

端口从 `config.toml` 的 `[server].port` 读取。

服务不再创建额外系统用户和用户组。systemd unit 会限制进程不提权、使用私有临时目录，并将系统目录设为只读；运行时数据只写入 `/var/lib/rss2cloud`。

HTTP 接口示例：

```bash
curl -d '{"tasks":["magnet:?xt=urn:btih:xx"],"cid":"12345","savepath":"文件夹名称"}' \
  -H "Content-Type: application/json" \
  -X POST http://127.0.0.1:8115/add
```

## 代理注意事项

如果 `node-site-config.json` 中启用了 `httpsAgent`，当前代码会使用 `http://127.0.0.1:10809` 作为代理地址。systemd 服务环境下，这要求代理服务也运行在同一台机器的 `127.0.0.1:10809`。

## 卸载

卸载服务和二进制，保留配置、cookies 和数据库：

```bash
curl -fsSL https://raw.githubusercontent.com/zhifengle/rss2cloud/main/scripts/install-release.sh | sudo bash -s -- uninstall
```

同时删除配置、cookies 和数据库：

```bash
curl -fsSL https://raw.githubusercontent.com/zhifengle/rss2cloud/main/scripts/install-release.sh | sudo bash -s -- purge
```
