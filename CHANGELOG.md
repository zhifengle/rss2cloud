# v0.2.6

## 变动

- 适配 AniBT RSS：缺少有效 `magneturi` 时，使用有效的 40 位十六进制 `infohash` 生成磁力链。
- AniBT 的 enclosure 回退仅接受磁力链，避免将 `.torrent` 下载地址作为普通 HTTP 下载提交给 115。
- 修复安装脚本在 `set -u` 模式下退出清理时访问已超出作用域的临时目录变量的问题。
- 将 systemd 的 `StartLimitIntervalSec` 和 `StartLimitBurst` 移至 `[Unit]`，使服务启动频率限制正确生效。
