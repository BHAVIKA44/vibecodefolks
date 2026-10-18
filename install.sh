#!/bin/sh
set -eu

repo=BHAVIKA44/vibecodefolks
version=${VIBECODEFOLKS_VERSION:-}
install_dir=${VIBECODEFOLKS_INSTALL_DIR:-"$HOME/.local/bin"}
command -v git >/dev/null 2>&1 || { echo 'Git is required for repository analysis.' >&2; exit 1; }
command -v curl >/dev/null 2>&1 || { echo 'curl is required.' >&2; exit 1; }
case $(uname -s) in Darwin) os=darwin ;; Linux) os=linux ;; *) echo 'Unsupported operating system.' >&2; exit 1 ;; esac
case $(uname -m) in x86_64|amd64) arch=amd64 ;; arm64|aarch64) arch=arm64 ;; *) echo 'Unsupported CPU architecture.' >&2; exit 1 ;; esac
if [ -z "$version" ]; then
  version=$(curl -fsSL "https://api.github.com/repos/$repo/releases/latest" | sed -n 's/^[[:space:]]*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)
fi
printf '%s\n' "$version" | LC_ALL=C grep -Eq '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$' || { echo 'Invalid stable release version.' >&2; exit 1; }
asset="vibecodefolks_${version}_${os}_${arch}.tar.gz"
url="https://github.com/$repo/releases/download/$version"
tmp=$(mktemp -d "${TMPDIR:-/tmp}/vibecodefolks-install.XXXXXXXX")
staged=
cleanup() { [ -z "$staged" ] || rm -f "$staged"; rm -rf "$tmp"; }
trap cleanup EXIT HUP INT TERM
curl -fsSL "$url/$asset" -o "$tmp/$asset"
curl -fsSL "$url/checksums.txt" -o "$tmp/checksums.txt"
expected=$(awk -v file="$asset" '$2 == file {print $1; count++} END {if (count != 1) exit 1}' "$tmp/checksums.txt") || { echo 'Missing or duplicate checksum.' >&2; exit 1; }
printf '%s\n' "$expected" | LC_ALL=C grep -Eq '^[0-9a-f]{64}$' || { echo 'Invalid checksum.' >&2; exit 1; }
if command -v sha256sum >/dev/null 2>&1; then actual=$(sha256sum "$tmp/$asset" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then actual=$(shasum -a 256 "$tmp/$asset" | awk '{print $1}')
else echo 'A SHA-256 tool is required.' >&2; exit 1; fi
[ "$expected" = "$actual" ] || { echo 'Checksum verification failed.' >&2; exit 1; }
[ "$(tar -tzf "$tmp/$asset")" = 'vibecodefolks' ] || { echo 'Unexpected archive contents.' >&2; exit 1; }
mkdir -p "$install_dir"
staged=$(mktemp "$install_dir/.vibecodefolks.XXXXXXXX")
tar -xOzf "$tmp/$asset" vibecodefolks > "$staged"
[ -s "$staged" ] || { echo 'Empty executable.' >&2; exit 1; }
chmod 755 "$staged"
mv -f "$staged" "$install_dir/vibecodefolks"
staged=
printf 'Installed %s to %s/vibecodefolks\n' "$version" "$install_dir"
case ":$PATH:" in *":$install_dir:"*) ;; *) printf 'Add %s to your PATH to run vibecodefolks.\n' "$install_dir" ;; esac
