#!/bin/sh
set -eu
version=${1:?expected version}; dest=${2:?expected output directory}
printf '%s\n' "$version" | LC_ALL=C grep -Eq '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$' || { echo 'Invalid version' >&2; exit 1; }
mkdir -p "$dest"
dest=$(cd "$dest" && pwd)
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT HUP INT TERM
for target in darwin_arm64 darwin_amd64 linux_arm64 linux_amd64 windows_amd64; do
 os=${target%_*}; arch=${target#*_}; name=vibecodefolks
 [ "$os" != windows ] || name=vibecodefolks.exe
 CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath -ldflags "-s -w -X main.version=$version" -o "$tmp/$name" .
 asset="vibecodefolks_${version}_${target}"
 if [ "$os" = windows ]; then
   (cd "$tmp" && TZ=UTC touch -t 202001010000 "$name" && zip -X -q "$dest/$asset.zip" "$name")
 else
   tar --sort=name --mtime='UTC 2020-01-01' --owner=0 --group=0 --numeric-owner -C "$tmp" -cf - "$name" | gzip -n > "$dest/$asset.tar.gz"
 fi
 rm -f "$tmp/$name"
done
(cd "$dest" && sha256sum vibecodefolks_*.tar.gz vibecodefolks_*.zip > checksums.txt && sha256sum -c checksums.txt)
