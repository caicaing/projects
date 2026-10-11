#!/usr/bin/env bash
# Public single-host deployment. Keep credentials out of shell arguments.
set -euo pipefail
umask 077

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
CONFIG_DIR="$ROOT/.taskflow-deploy"
INSTALL_DIR='/opt/taskflow'
SYSTEM_CONFIG='/etc/taskflow'
DATA_DIR='/var/lib/taskflow'
NODE_BIN=''
CADDY_BIN=''
GO_BIN=''
NPM_BIN=''
DOMAIN_ARG=''
USER_ARG=''
ACTION='up'
TEMP_DIR=''
DOMAIN=''
USERNAME=''
PASSWORD=''
TOKEN=''

fail() {
  printf '部署失败：%s\n' "$*" >&2
  exit 1
}

usage() {
  cat <<'HELP'
用法：sudo bash scripts/deploy.sh [--domain 域名] [--user 账号]
      sudo bash scripts/deploy.sh --status | --logs | --stop

首次运行会询问域名和登录密码，账号默认 admin，后端 Token 自动生成。
前后端配置分别保存在 .taskflow-deploy/backend.yaml、frontend.yaml。
后续直接编辑配置并重新运行，不读取环境变量。
非交互部署请预先准备这两份配置和 .taskflow-deploy/Caddyfile。

要求：Linux、systemd；Debian 13（含 13.2）自动安装缺失或过旧的依赖。
自动安装支持 x86_64/ARM64，包含 Go、Node.js 24 LTS、npm 和 Caddy。
公网域名已解析到此服务器，
安全组/防火墙放行 TCP 80、443；宿主机这些端口未被其他服务占用。
Caddy 自动签发和续期 HTTPS 证书，systemd 管理进程和开机启动。
脚本安装到 /opt/taskflow、/etc/taskflow 和 /var/lib/taskflow。
HELP
}

parse_args() {
  while (($#)); do
    case "$1" in
      --domain|--user)
        (($# >= 2)) || fail "$1 缺少参数"
        if [[ "$1" == --domain ]]; then
          DOMAIN_ARG=$2
        else
          USER_ARG=$2
        fi
        shift 2
        ;;
      --status|--logs|--stop)
        [[ "$ACTION" == up ]] || fail '操作参数只能指定一个'
        ACTION=${1#--}
        shift
        ;;
      -h|--help) usage; exit 0 ;;
      *) fail "未知参数：$1" ;;
    esac
  done
}

check_environment() {
  [[ $(uname -s) == Linux ]] || fail '该脚本用于 Linux 服务器'
  [[ $EUID -eq 0 ]] || fail '请使用 sudo 运行，以安装 systemd 服务'
  command -v systemctl >/dev/null || fail '需要 systemd'
  [[ -d /run/systemd/system ]] || fail 'systemd 未运行'
  [[ "$ACTION" == up ]] || return 0
  [[ -f "$ROOT/scripts/install-deps.sh" ]] || fail '依赖安装脚本缺失'
  . "$ROOT/scripts/install-deps.sh"
  ensure_dependencies
  export PATH="$DEPENDENCY_ROOT/bin:$PATH"
  hash -r
  local command
  for command in go node npm caddy useradd runuser install; do
    command -v "$command" >/dev/null || fail "请先安装 $command"
  done
  NODE_BIN=$(command -v node)
  CADDY_BIN=$(command -v caddy)
  GO_BIN=$(command -v go)
  NPM_BIN=$(command -v npm)
  [[ "$NODE_BIN" =~ ^/[a-zA-Z0-9_./+-]+$ \
    && "$CADDY_BIN" =~ ^/[a-zA-Z0-9_./+-]+$ ]] \
    || fail 'Node.js 和 Caddy 需要安装在标准的绝对路径中'
  [[ -f "$ROOT/deploy/taskflow.service" \
    && -f "$ROOT/deploy/taskflow-caddy.service" ]] \
    || fail 'systemd 服务模板缺失'
}

validate_config() {
  local label
  local labels=()
  [[ ${#DOMAIN} -le 253 && "$DOMAIN" == *.* ]] \
    || fail '请填写公网域名，例如 taskflow.example.com'
  [[ "$DOMAIN" != *[!a-zA-Z0-9.-]* ]] \
    || fail '域名不能包含协议、端口、路径或特殊字符'
  [[ "$DOMAIN" != .* && "$DOMAIN" != *. \
    && "$DOMAIN" != *..* ]] || fail '域名格式不正确'
  [[ ! "$DOMAIN" =~ ^[0-9.]+$ ]] || fail '请使用域名而不是 IP'
  IFS='.' read -r -a labels <<< "$DOMAIN"
  for label in "${labels[@]}"; do
    [[ ${#label} -le 63 && "$label" != -* && "$label" != *- ]] \
      || fail '域名格式不正确'
  done
  [[ -n "$USERNAME" && ${#USERNAME} -le 128 \
    && "$USERNAME" != *$'\n'* \
    && "$USERNAME" != *$'\r'* ]] || fail '登录账号格式不正确'
  [[ ${#PASSWORD} -ge 12 \
    && ${#PASSWORD} -le 1024 \
    && "$PASSWORD" != *$'\n'* \
    && "$PASSWORD" != *$'\r'* ]] \
    || fail '登录密码需为 12–1024 字符，不能包含换行'
  [[ ${#TOKEN} -ge 16 && ${#TOKEN} -le 4096 \
    && "$TOKEN" != *[![:graph:]]* ]] \
    || fail '后端 Token 至少 16 字符且不能含空白字符'
}

load_config() {
  [[ -d "$CONFIG_DIR" && ! -L "$CONFIG_DIR" ]] \
    || fail '.taskflow-deploy 必须是普通目录'
  chmod 700 "$CONFIG_DIR"
  local file
  for file in backend.yaml frontend.yaml Caddyfile; do
    [[ -f "$CONFIG_DIR/$file" && ! -L "$CONFIG_DIR/$file" ]] \
      || fail "部署配置缺失：$file，请按 README 准备分开的配置文件"
    chmod 600 "$CONFIG_DIR/$file"
  done
  [[ -z "$DOMAIN_ARG" && -z "$USER_ARG" ]] \
    || fail '已有部署配置，请直接修改 YAML 和 Caddyfile 后重新运行'
}

prompt_password() {
  local confirm
  [[ -t 0 ]] || fail '非交互部署请预先准备 .taskflow-deploy/ 配置'
  IFS= read -r -s -p '设置登录密码（至少 12 字符）：' PASSWORD
  printf '\n'
  IFS= read -r -s -p '再次输入登录密码：' confirm
  printf '\n'
  [[ "$PASSWORD" == "$confirm" ]] || fail '两次密码不一致'
}

yaml_quote() {
  local value=${1//\'/\'\'}
  printf "'%s'" "$value"
}

new_config() {
  DOMAIN=$DOMAIN_ARG
  USERNAME=${USER_ARG:-admin}
  if [[ -z "$DOMAIN" ]]; then
    [[ -t 0 ]] || fail '首次部署需 --domain 或交互输入域名'
    IFS= read -r -p '管理台域名（不含 https://）：' DOMAIN
  fi
  prompt_password
  TOKEN=$(od -An -N32 -tx1 /dev/urandom | tr -d ' \n')
  validate_config
  TEMP_DIR=$(mktemp -d "$ROOT/.taskflow-deploy.tmp.XXXXXX")
  cat > "$TEMP_DIR/backend.yaml" <<YAML
backend:
  addr: "127.0.0.1:8081"
  database: "$DATA_DIR/taskflow.db"
  token: $(yaml_quote "$TOKEN")
scheduler:
  workers: 8
  history_limit: 500
  task_timeout: 30s
  notification_timeout: 10s
YAML
  cat > "$TEMP_DIR/frontend.yaml" <<YAML
web:
  host: "127.0.0.1"
  port: 8080
  origin: "https://$DOMAIN"
  username: $(yaml_quote "$USERNAME")
  password: $(yaml_quote "$PASSWORD")
backend:
  url: "http://127.0.0.1:8081"
  token: $(yaml_quote "$TOKEN")
YAML
  cat > "$TEMP_DIR/Caddyfile" <<CADDY
$DOMAIN {
    header Strict-Transport-Security "max-age=31536000"
    reverse_proxy 127.0.0.1:8080
}
CADDY
  chmod 600 "$TEMP_DIR/backend.yaml" "$TEMP_DIR/frontend.yaml" \
    "$TEMP_DIR/Caddyfile"
  [[ ! -e "$CONFIG_DIR" && ! -L "$CONFIG_DIR" ]] \
    || fail '配置目录已存在，请重新运行脚本'
  mv -T -- "$TEMP_DIR" "$CONFIG_DIR"
  TEMP_DIR=''
  PASSWORD=''
  TOKEN=''
}

build_application() {
  TEMP_DIR=$(mktemp -d "$ROOT/.taskflow-deploy.tmp.XXXXXX")
  local app="$TEMP_DIR/app"
  mkdir -p "$app/backend/bin" "$app/frontend" "$app/scripts"
  (
    cd "$ROOT/backend"
    "$GO_BIN" build -trimpath -o "$app/backend/bin/taskflow" ./cmd/taskflow
  )
  cp "$ROOT/frontend/package.json" "$ROOT/frontend/package-lock.json" \
    "$ROOT/frontend/"*.mjs "$app/frontend/"
  cp -R "$ROOT/frontend/public" "$ROOT/frontend/src" "$app/frontend/"
  cp "$ROOT/scripts/dev.mjs" "$app/scripts/"
  (
    cd "$app/frontend"
    "$NPM_BIN" ci --omit=dev --ignore-scripts --no-audit --no-fund
  )
  "$NODE_BIN" "$app/scripts/dev.mjs" --check \
    --backend-config "$CONFIG_DIR/backend.yaml" \
    --frontend-config "$CONFIG_DIR/frontend.yaml"
  (
    cd "$app/frontend"
    "$NODE_BIN" --input-type=module - "$CONFIG_DIR/frontend.yaml" \
      "$CONFIG_DIR/backend.yaml" <<'JS'
import { isAbsolute, resolve } from 'node:path';
import { webConfig } from './auth.mjs';
import { loadConfig, readConfig } from './config.mjs';
const config = webConfig(loadConfig(process.argv[2]));
const database = readConfig(process.argv[3]).backend.database;
if (config.origin.protocol !== 'https:' || config.host !== '127.0.0.1') {
  throw new Error('部署要求 HTTPS origin，web.host 为 127.0.0.1');
}
if (typeof database !== 'string' || !isAbsolute(database)
  || !resolve(database).startsWith('/var/lib/taskflow/')) {
  throw new Error('部署数据库必须使用 /var/lib/taskflow/ 下的绝对路径');
}
JS
  )
  "$CADDY_BIN" validate --config "$CONFIG_DIR/Caddyfile" \
    --adapter caddyfile
}

install_services() {
  if ! id taskflow >/dev/null 2>&1; then
    useradd --system --user-group --home-dir "$DATA_DIR" \
      --shell /usr/sbin/nologin taskflow
  fi
  runuser -u taskflow -- "$NODE_BIN" --version >/dev/null \
    || fail 'taskflow 用户无法执行 Node.js，请使用系统级安装'
  runuser -u taskflow -- "$CADDY_BIN" version >/dev/null \
    || fail 'taskflow 用户无法执行 Caddy，请使用系统级安装'
  install -d -m 700 -o taskflow -g taskflow "$DATA_DIR" "$DATA_DIR/caddy"
  install -d -m 750 -o root -g taskflow "$SYSTEM_CONFIG"
  # All validation and builds finish before stopping the existing services.
  if systemctl cat taskflow-caddy.service >/dev/null 2>&1; then
    systemctl stop taskflow-caddy.service
  fi
  if systemctl cat taskflow.service >/dev/null 2>&1; then
    systemctl stop taskflow.service
  fi
  install -d -m 755 "$INSTALL_DIR"
  cp -R "$TEMP_DIR/app/." "$INSTALL_DIR/"
  chmod -R u=rwX,go=rX "$INSTALL_DIR"
  local file
  for file in backend.yaml frontend.yaml Caddyfile; do
    install -m 640 -o root -g taskflow "$CONFIG_DIR/$file" \
      "$SYSTEM_CONFIG/$file"
  done
  sed "s|@NODE@|$NODE_BIN|g" "$ROOT/deploy/taskflow.service" \
    > "$TEMP_DIR/taskflow.service"
  sed "s|@CADDY@|$CADDY_BIN|g" "$ROOT/deploy/taskflow-caddy.service" \
    > "$TEMP_DIR/taskflow-caddy.service"
  install -m 644 "$TEMP_DIR/"*.service /etc/systemd/system/
  systemctl daemon-reload
  systemctl enable taskflow.service taskflow-caddy.service
  systemctl start taskflow.service taskflow-caddy.service
}

probe_services() {
  systemctl is-active --quiet taskflow.service taskflow-caddy.service \
    || return 1
  (
    cd "$INSTALL_DIR/frontend"
    "$NODE_BIN" --input-type=module - \
      --config "$SYSTEM_CONFIG/frontend.yaml" 2>/dev/null <<'JS'
import https from 'node:https';
import { webConfig } from './auth.mjs';
try {
  const config = webConfig();
  const api = await fetch(`${config.backend.origin}/api/catalog`, {
    headers: { Authorization: `Bearer ${config.token}` },
    signal: AbortSignal.timeout(3000),
  });
  if (!api.ok) process.exit(1);
  await new Promise((resolve, reject) => {
    const domain = config.origin.hostname;
    const req = https.get({ hostname: '127.0.0.1', port: 443,
      servername: domain, path: '/login.html', headers: { Host: domain },
      signal: AbortSignal.timeout(3000) },
    (res) => {
      res.resume();
      if (res.statusCode === 200) resolve();
      else reject(new Error('gateway unavailable'));
    });
    req.on('error', reject);
  });
} catch { process.exit(1); }
JS
  )
}

wait_for_services() {
  local deadline=$((SECONDS + 120))
  printf '正在等待后端、登录网关及 HTTPS 证书就绪（最多 120 秒）…\n'
  while ((SECONDS < deadline)); do
    if probe_services; then
      return
    fi
    sleep 2
  done
  printf '服务保持运行，可查看日志排查 DNS、端口或证书申请问题：\n' >&2
  printf 'sudo bash %q --logs\n' "$ROOT/scripts/deploy.sh" >&2
  fail '服务或 HTTPS 尚未就绪；修复后可再次运行脚本'
}

cleanup() {
  if [[ -n "$TEMP_DIR" && -d "$TEMP_DIR" ]]; then
    rm -rf -- "$TEMP_DIR"
  fi
}

main() {
  parse_args "$@"
  check_environment
  case "$ACTION" in
    status)
      systemctl status taskflow.service taskflow-caddy.service --no-pager
      return ;;
    logs)
      journalctl -u taskflow.service -u taskflow-caddy.service \
        --no-pager -n 100
      return ;;
    stop)
      systemctl stop taskflow-caddy.service taskflow.service
      return ;;
  esac
  if [[ -e "$CONFIG_DIR" || -L "$CONFIG_DIR" ]]; then
    load_config
  else
    [[ "$ACTION" == up ]] || fail '尚未部署，请先运行部署脚本'
    new_config
  fi
  build_application
  install_services
  wait_for_services
  printf '部署完成，访问地址和账号见 .taskflow-deploy/frontend.yaml。\n'
  printf '修改配置后重新运行本脚本即可生效。\n'
}

trap cleanup EXIT
main "$@"
