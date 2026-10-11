#!/usr/bin/env bash
# Install missing runtimes without starting an additional Caddy service.
set -euo pipefail

DEPENDENCY_ROOT='/opt/taskflow-runtimes'
DEPENDENCY_TEMP=''
DEPENDENCY_ARCH=''
DEPENDENCY_NODE_ARCH=''

deps_fail() {
  printf '依赖安装失败：%s\n' "$*" >&2
  exit 1
}

deps_version_at_least() {
  local version=${1#v}
  [[ "$version" =~ ^([0-9]+)\.([0-9]+)(\.|$) ]] || return 1
  local major=${BASH_REMATCH[1]}
  local minor=${BASH_REMATCH[2]}
  ((major > $2 || (major == $2 && minor >= $3)))
}

deps_go_ready() {
  command -v go >/dev/null || return 1
  local version
  version=$(go version 2>/dev/null) || return 1
  version=${version#go version go}
  deps_version_at_least "${version%% *}" 1 23
}

deps_node_ready() {
  command -v node >/dev/null || return 1
  command -v npm >/dev/null || return 1
  local version
  version=$(node --version 2>/dev/null) || return 1
  deps_version_at_least "$version" 18 0 || return 1
  npm --version >/dev/null 2>&1
}

deps_caddy_ready() {
  command -v caddy >/dev/null || return 1
  local version
  version=$(caddy version 2>/dev/null) || return 1
  deps_version_at_least "${version%% *}" 2 0
}

deps_ready() {
  deps_go_ready && deps_node_ready && deps_caddy_ready || return 1
  local utility
  for utility in useradd runuser install; do
    command -v "$utility" >/dev/null || return 1
  done
}

deps_platform() {
  [[ -f /etc/os-release ]] || deps_fail '无法识别 Linux 发行版'
  local ID='' VERSION_ID=''
  . /etc/os-release
  [[ "$ID" == debian && "${VERSION_ID%%.*}" == 13 ]] \
    || deps_fail '自动安装面向 Debian 13（含 13.2）；其他系统请先安装依赖'
  command -v apt-get >/dev/null || deps_fail '需要 apt-get'
  case $(uname -m) in
    x86_64) DEPENDENCY_ARCH='amd64'; DEPENDENCY_NODE_ARCH='x64' ;;
    aarch64|arm64)
      DEPENDENCY_ARCH='arm64'; DEPENDENCY_NODE_ARCH='arm64' ;;
    *) deps_fail '自动安装支持 x86_64 和 ARM64' ;;
  esac
}

deps_download() {
  curl --fail --location --silent --show-error --retry 3 \
    --connect-timeout 15 --max-time 300 --proto '=https' \
    --proto-redir '=https' "$1" --output "$2"
}

deps_verify() {
  local digest='sha256sum'
  if [[ "$2" =~ ^[a-fA-F0-9]{128}$ ]]; then
    digest='sha512sum'
  elif [[ ! "$2" =~ ^[a-fA-F0-9]{64}$ ]]; then
    deps_fail '官方校验值格式错误'
  fi
  (
    cd "$DEPENDENCY_TEMP"
    printf '%s  %s\n' "$2" "$1" | "$digest" --check --status
  ) || deps_fail "下载文件校验失败：$1"
}

deps_link() {
  ln -sfn "$1" "$DEPENDENCY_ROOT/bin/$2"
}

deps_install_go() {
  printf '正在安装 Go 官方稳定版…\n'
  deps_download 'https://go.dev/dl/?mode=json' "$DEPENDENCY_TEMP/go.json"
  local filename checksum version
  local metadata
  metadata=$(python3 - "$DEPENDENCY_TEMP/go.json" "$DEPENDENCY_ARCH" \
    <<'PY'
import json
import re
import sys

with open(sys.argv[1]) as source:
    releases = json.load(source)
for release in releases:
    version = release['version']
    if not release['stable'] or not re.fullmatch(r'go\d+\.\d+\.\d+', version):
        continue
    for asset in release['files']:
        filename = f'{version}.linux-{sys.argv[2]}.tar.gz'
        if asset['filename'] == filename and asset['kind'] == 'archive':
            print(filename, asset['sha256'], version)
            sys.exit(0)
raise SystemExit('未找到当前架构的 Go 稳定版')
PY
  ) || deps_fail '读取 Go 官方版本信息失败'
  read -r filename checksum version <<< "$metadata"
  deps_download "https://go.dev/dl/$filename" "$DEPENDENCY_TEMP/$filename"
  deps_verify "$filename" "$checksum"
  tar -xzf "$DEPENDENCY_TEMP/$filename" -C "$DEPENDENCY_TEMP"
  chmod -R u=rwX,go=rX "$DEPENDENCY_TEMP/go"
  install -d -m 755 "$DEPENDENCY_ROOT/go"
  mv -T "$DEPENDENCY_TEMP/go" "$DEPENDENCY_ROOT/go/$version"
  deps_link "$DEPENDENCY_ROOT/go/$version/bin/go" go
}

deps_install_node() {
  printf '正在安装 Node.js 24 LTS（包含 npm）…\n'
  local base='https://nodejs.org/dist/latest-v24.x'
  local sums="$DEPENDENCY_TEMP/node-shasums.txt"
  deps_download "$base/SHASUMS256.txt" "$sums"
  local filename checksum
  read -r checksum filename < <(awk -v arch="$DEPENDENCY_NODE_ARCH" \
    '$2 ~ "^node-v24\\.[0-9]+\\.[0-9]+-linux-" arch "\\.tar\\.xz$" \
      { print $1, $2; exit }' "$sums") \
    || deps_fail '未找到当前架构的 Node.js 24 LTS'
  deps_download "$base/$filename" "$DEPENDENCY_TEMP/$filename"
  deps_verify "$filename" "$checksum"
  tar -xJf "$DEPENDENCY_TEMP/$filename" -C "$DEPENDENCY_TEMP"
  local directory=${filename%.tar.xz}
  chmod -R u=rwX,go=rX "$DEPENDENCY_TEMP/$directory"
  install -d -m 755 "$DEPENDENCY_ROOT/node"
  mv -T "$DEPENDENCY_TEMP/$directory" "$DEPENDENCY_ROOT/node/$directory"
  deps_link "$DEPENDENCY_ROOT/node/$directory/bin/node" node
  deps_link "$DEPENDENCY_ROOT/node/$directory/bin/npm" npm
}

deps_install_caddy() {
  printf '正在安装 Caddy 官方稳定版…\n'
  local metadata="$DEPENDENCY_TEMP/caddy.json"
  local api='https://api.github.com/repos/caddyserver/caddy/releases/latest'
  deps_download "$api" "$metadata"
  local version
  version=$(python3 - "$metadata" <<'PY'
import json
import re
import sys

with open(sys.argv[1]) as source:
    release = json.load(source)
version = release['tag_name']
if release['prerelease'] or not re.fullmatch(r'v2\.\d+\.\d+', version):
    raise SystemExit('未找到 Caddy 2 稳定版')
print(version[1:])
PY
  ) || deps_fail '读取 Caddy 官方版本信息失败'
  local base="https://github.com/caddyserver/caddy/releases/download"
  local filename="caddy_${version}_linux_${DEPENDENCY_ARCH}.tar.gz"
  local sums="$DEPENDENCY_TEMP/caddy-checksums.txt"
  deps_download "$base/v$version/caddy_${version}_checksums.txt" "$sums"
  local checksum
  checksum=$(awk -v filename="$filename" '$2 == filename {print $1}' "$sums")
  deps_download "$base/v$version/$filename" "$DEPENDENCY_TEMP/$filename"
  deps_verify "$filename" "$checksum"
  tar -xzf "$DEPENDENCY_TEMP/$filename" -C "$DEPENDENCY_TEMP" caddy
  install -d -m 755 "$DEPENDENCY_ROOT/caddy/$version"
  install -m 755 "$DEPENDENCY_TEMP/caddy" \
    "$DEPENDENCY_ROOT/caddy/$version/caddy"
  deps_link "$DEPENDENCY_ROOT/caddy/$version/caddy" caddy
}

deps_cleanup() {
  if [[ -n "$DEPENDENCY_TEMP" && -d "$DEPENDENCY_TEMP" ]]; then
    rm -rf -- "$DEPENDENCY_TEMP"
  fi
}

ensure_dependencies() (
  [[ $(uname -s) == Linux && $EUID -eq 0 ]] \
    || deps_fail '请在 Linux 服务器上使用 sudo 运行'
  umask 022
  export PATH="$DEPENDENCY_ROOT/bin:$PATH"
  if deps_ready; then
    printf '已有 Go、Node.js、npm 和 Caddy 可用，跳过安装。\n'
    return
  fi
  deps_platform
  printf '正在安装下载工具与系统依赖…\n'
  apt-get update
  DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    ca-certificates curl python3 tar xz-utils passwd util-linux coreutils git
  install -d -m 755 "$DEPENDENCY_ROOT" "$DEPENDENCY_ROOT/bin"
  DEPENDENCY_TEMP=$(mktemp -d "$DEPENDENCY_ROOT/.install.XXXXXX")
  trap deps_cleanup EXIT
  if ! deps_go_ready; then deps_install_go; fi
  if ! deps_node_ready; then deps_install_node; fi
  if ! deps_caddy_ready; then deps_install_caddy; fi
  hash -r
  deps_ready || deps_fail '安装后的依赖版本检查失败'
)

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  ensure_dependencies
fi
