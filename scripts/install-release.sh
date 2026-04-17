#!/usr/bin/env bash

set -Eeuo pipefail

APP_NAME='rss2cloud'
REPO='zhifengle/rss2cloud'
BIN_PATH='/usr/local/bin/rss2cloud'
DATA_DIR='/var/lib/rss2cloud'
SERVICE_USER='rss2cloud'
SERVICE_PORT='8115'
SERVICE_FILE='/etc/systemd/system/rss2cloud.service'

info() {
  echo "info: $*"
}

error() {
  echo "error: $*" >&2
}

die() {
  error "$*"
  exit 1
}

require_root() {
  if [ "$(id -u)" -ne 0 ]; then
    die 'This script must be run as root.'
  fi
}

require_linux_systemd() {
  [ "$(uname -s)" = 'Linux' ] || die 'Only Linux is supported.'
  command -v systemctl >/dev/null 2>&1 || die 'systemctl is required.'
  [ -d /run/systemd/system ] || die 'systemd is not running on this system.'
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "$1 is required."
}

detect_arch() {
  case "$(uname -m)" in
    x86_64 | amd64)
      echo 'amd64'
      ;;
    *)
      die 'Only linux amd64 release assets are supported by the current release workflow.'
      ;;
  esac
}

latest_version() {
  local api tag
  api="https://api.github.com/repos/${REPO}/releases/latest"
  if ! tag="$(curl -fsSL "$api" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"; then
    die 'Failed to request latest release version.'
  fi
  [ -n "$tag" ] || die 'Failed to resolve latest release version.'
  echo "$tag"
}

install_service_user() {
  local nologin_shell

  require_command getent
  if ! getent group "$SERVICE_USER" >/dev/null 2>&1; then
    require_command groupadd
    groupadd --system "$SERVICE_USER"
    info "created system group: ${SERVICE_USER}"
  fi

  if id "$SERVICE_USER" >/dev/null 2>&1; then
    return
  fi

  require_command useradd
  nologin_shell="$(command -v nologin || true)"
  if [ -z "$nologin_shell" ]; then
    nologin_shell='/usr/sbin/nologin'
  fi

  useradd --system --gid "$SERVICE_USER" --home-dir "$DATA_DIR" --shell "$nologin_shell" "$SERVICE_USER"
  info "created system user: ${SERVICE_USER}"
}

install_data_dir() {
  install -d -m 0755 -o "$SERVICE_USER" -g "$SERVICE_USER" "$DATA_DIR"
  info "installed: ${DATA_DIR}"
}

download_binary() {
  local version arch url tmp_file

  version="$1"
  arch="$2"
  tmp_file="$3"
  url="https://github.com/${REPO}/releases/download/${version}/${APP_NAME}-${version}-linux-${arch}-musl.tar.gz"

  info "downloading: ${url}"
  curl -fL --retry 3 --retry-delay 3 -o "$tmp_file" "$url"
}

install_binary() {
  local archive tmp_dir binary

  archive="$1"
  tmp_dir="$2"
  tar -xzf "$archive" -C "$tmp_dir"
  binary="${tmp_dir}/${APP_NAME}"
  [ -f "$binary" ] || die "Archive does not contain ${APP_NAME}."
  install -m 0755 -o root -g root "$binary" "$BIN_PATH"
  info "installed: ${BIN_PATH}"
}

install_systemd_service() {
  cat >"$SERVICE_FILE" <<SERVICE
[Unit]
Description=rss2cloud server
Documentation=https://github.com/${REPO}
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${SERVICE_USER}
Group=${SERVICE_USER}
WorkingDirectory=${DATA_DIR}
ExecStart=${BIN_PATH} server --port ${SERVICE_PORT}
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
SERVICE

  chmod 0644 "$SERVICE_FILE"
  systemctl daemon-reload
  info "installed: ${SERVICE_FILE}"
}

print_next_steps() {
  cat <<EOF

rss2cloud is installed.

Put runtime files in ${DATA_DIR}:
  - .cookies
  - rss.json
  - node-site-config.json

Then start the service:
  systemctl enable --now rss2cloud

Useful commands:
  systemctl status rss2cloud
  journalctl -u rss2cloud -f
EOF
}

main() {
  local arch version tmp_dir tmp_file

  require_root
  require_linux_systemd
  require_command curl
  require_command sed
  require_command install
  require_command tar

  arch="$(detect_arch)"
  version="$(latest_version)"
  tmp_dir="$(mktemp -d)"
  tmp_file="${tmp_dir}/${APP_NAME}.tar.gz"
  trap 'rm -rf "$tmp_dir"' EXIT

  download_binary "$version" "$arch" "$tmp_file"
  install_service_user
  install_data_dir
  install_binary "$tmp_file" "$tmp_dir"
  install_systemd_service
  print_next_steps
}

main "$@"
