#!/bin/sh
set -eu
root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT HUP INT TERM
mkdir -p "$tmp/bin" "$tmp/server" "$tmp/content" "$tmp/home"
case $(uname -s) in Darwin) os=darwin ;; Linux) os=linux ;; *) echo 'installer test unsupported OS' >&2; exit 1;; esac
case $(uname -m) in x86_64|amd64) arch=amd64 ;; arm64|aarch64) arch=arm64 ;; *) exit 1 ;; esac
asset="vibecodefolks_v0.1.0_${os}_${arch}.tar.gz"
printf 'fixture executable\n' > "$tmp/content/vibecodefolks"
tar -czf "$tmp/server/$asset" -C "$tmp/content" vibecodefolks
if command -v sha256sum >/dev/null 2>&1; then digest=$(sha256sum "$tmp/server/$asset" | awk '{print $1}'); else digest=$(shasum -a 256 "$tmp/server/$asset" | awk '{print $1}'); fi
printf '%s  %s\n' "$digest" "$asset" > "$tmp/server/checksums.txt"
cat > "$tmp/bin/curl" <<'MOCK'
#!/bin/sh
set -eu
out=
while [ "$#" -gt 0 ]; do case "$1" in -o) out=$2; shift 2;; -fsSL) shift;; https://*) url=$1; shift;; *) exit 1;; esac; done
case "$url" in */releases/latest) printf '{\n  "tag_name": "v0.1.0"\n}\n';; *) cp "$FIXTURE_DIR/${url##*/}" "$out";; esac
MOCK
chmod +x "$tmp/bin/curl"
export FIXTURE_DIR="$tmp/server" HOME="$tmp/home" PATH="$tmp/bin:$PATH"
sh "$root/install.sh"
cmp "$tmp/content/vibecodefolks" "$HOME/.local/bin/vibecodefolks"
printf 'tampered\n' >> "$tmp/server/$asset"
if VIBECODEFOLKS_VERSION=v0.1.0 sh "$root/install.sh" >/dev/null 2>&1; then echo 'tampered archive was accepted' >&2; exit 1; fi
cmp "$tmp/content/vibecodefolks" "$HOME/.local/bin/vibecodefolks"
if VIBECODEFOLKS_VERSION='v1.0.0/../../bad' sh "$root/install.sh" >/dev/null 2>&1; then echo 'invalid version was accepted' >&2; exit 1; fi
# Restore the verified archive, then reject missing and duplicate checksum entries.
tar -czf "$tmp/server/$asset" -C "$tmp/content" vibecodefolks
if command -v sha256sum >/dev/null 2>&1; then digest=$(sha256sum "$tmp/server/$asset" | awk '{print $1}'); else digest=$(shasum -a 256 "$tmp/server/$asset" | awk '{print $1}'); fi
: > "$tmp/server/checksums.txt"
if VIBECODEFOLKS_VERSION=v0.1.0 sh "$root/install.sh" >/dev/null 2>&1; then echo 'missing checksum was accepted' >&2; exit 1; fi
printf '%s  %s\n%s  %s\n' "$digest" "$asset" "$digest" "$asset" > "$tmp/server/checksums.txt"
if VIBECODEFOLKS_VERSION=v0.1.0 sh "$root/install.sh" >/dev/null 2>&1; then echo 'duplicate checksum was accepted' >&2; exit 1; fi
cmp "$tmp/content/vibecodefolks" "$HOME/.local/bin/vibecodefolks"
echo 'Installer tests passed'
