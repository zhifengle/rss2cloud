#!/usr/bin/env bash

set -Eeuo pipefail

APP_NAME='rss2cloud'
REPO='zhifengle/rss2cloud'
BIN_PATH='/usr/local/bin/rss2cloud'
DATA_DIR='/var/lib/rss2cloud'
CONFIG_FILE="${DATA_DIR}/config.toml"
COOKIES_FILE="${DATA_DIR}/.cookies"
DB_FILE="${DATA_DIR}/db.sqlite"
SERVICE_FILE='/etc/systemd/system/rss2cloud.service'
SERVICE_PORT='8115'

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

usage() {
  cat <<EOF
Usage:
  install-release.sh [install|update|uninstall|purge]

Commands:
  install     Install or update rss2cloud. This is the default command.
  update      Same as install.
  uninstall   Stop service and remove binary/service files. Keep config and data.
  purge       Uninstall and remove config and data.

Examples:
  curl -fsSL https://raw.githubusercontent.com/${REPO}/main/scripts/install-release.sh | sudo bash
  curl -fsSL https://raw.githubusercontent.com/${REPO}/main/scripts/install-release.sh | sudo bash -s -- uninstall
EOF
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

install_config_dir() {
  install -d -m 0700 "$DATA_DIR"
}

install_default_config() {
  if [ -e "$CONFIG_FILE" ]; then
    chmod 0640 "$CONFIG_FILE"
    info "kept existing: ${CONFIG_FILE}"
  else
    cat >"$CONFIG_FILE" <<EOF
[auth]
cookies_file = ".cookies"

[server]
port = ${SERVICE_PORT}

[database]
path = "db.sqlite"
EOF
    chmod 0640 "$CONFIG_FILE"
    info "installed: ${CONFIG_FILE}"
  fi

  if [ -e "$COOKIES_FILE" ]; then
    chmod 0600 "$COOKIES_FILE"
    info "kept existing: ${COOKIES_FILE}"
  else
    : >"$COOKIES_FILE"
    chmod 0600 "$COOKIES_FILE"
    info "created: ${COOKIES_FILE}"
  fi

  if [ -e "$DB_FILE" ]; then
    chmod 0600 "$DB_FILE"
    info "kept existing: ${DB_FILE}"
  fi
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
  cat >"$SERVICE_FILE" <<EOF
[Unit]
Description=rss2cloud server
Documentation=https://github.com/${REPO}
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=${DATA_DIR}
ExecStart=${BIN_PATH} server
Restart=on-failure
RestartSec=10s
StartLimitIntervalSec=300
StartLimitBurst=2
UMask=0077
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ReadWritePaths=${DATA_DIR}

[Install]
WantedBy=multi-user.target
EOF

  chmod 0644 "$SERVICE_FILE"
  systemctl daemon-reload
  info "installed: ${SERVICE_FILE}"
}

print_install_next_steps() {
  cat <<EOF

rss2cloud is installed.

Config:
  ${CONFIG_FILE}

Cookies:
  ${COOKIES_FILE}

Start service after cookies are ready:
  systemctl enable --now ${APP_NAME}

Useful commands:
  systemctl status ${APP_NAME}
  journalctl -u ${APP_NAME} -f
  systemctl restart ${APP_NAME}
EOF
}

install_release() {
  local arch version tmp_dir tmp_file

  require_linux_systemd
  require_command curl
  require_command sed
  require_command install
  require_command tar
  require_command mktemp

  arch="$(detect_arch)"
  version="$(latest_version)"
  tmp_dir="$(mktemp -d)"
  tmp_file="${tmp_dir}/${APP_NAME}.tar.gz"
  trap 'rm -rf "$tmp_dir"' EXIT

  download_binary "$version" "$arch" "$tmp_file"
  install_config_dir
  install_default_config
  install_binary "$tmp_file" "$tmp_dir"
  install_systemd_service
  print_install_next_steps
}

stop_service_if_present() {
  if [ -f "$SERVICE_FILE" ]; then
    systemctl stop "$APP_NAME" >/dev/null 2>&1 || true
    systemctl disable "$APP_NAME" >/dev/null 2>&1 || true
  fi
}

uninstall_release() {
  require_linux_systemd

  stop_service_if_present

  if [ -f "$SERVICE_FILE" ]; then
    rm -f "$SERVICE_FILE"
    info "removed: ${SERVICE_FILE}"
  fi

  systemctl daemon-reload
  systemctl reset-failed "$APP_NAME" >/dev/null 2>&1 || true

  if [ -f "$BIN_PATH" ]; then
    rm -f "$BIN_PATH"
    info "removed: ${BIN_PATH}"
  fi

  cat <<EOF

rss2cloud is uninstalled.

Kept config and data:
  ${DATA_DIR}

Remove them too:
  curl -fsSL https://raw.githubusercontent.com/${REPO}/main/scripts/install-release.sh | sudo bash -s -- purge
EOF
}

purge_release() {
  uninstall_release

  if [ -d "$DATA_DIR" ]; then
    rm -rf "$DATA_DIR"
    info "removed: ${DATA_DIR}"
  fi
}

main() {
  local command

  command="${1:-install}"
  case "$command" in
    -h | --help | help)
      usage
      ;;
    install | update)
      require_root
      install_release
      ;;
    uninstall)
      require_root
      uninstall_release
      ;;
    purge)
      require_root
      purge_release
      ;;
    *)
      usage
      die "unknown command: ${command}"
      ;;
  esac
}

main "$@"
