#!/usr/bin/env python3
import hashlib, pathlib, sys
checksums, asset = map(pathlib.Path, sys.argv[1:])
entries = [line.split() for line in checksums.read_text().splitlines()]
matches = [entry[0] for entry in entries if len(entry) == 2 and entry[1] == asset.name]
if len(matches) != 1 or hashlib.sha256(asset.read_bytes()).hexdigest() != matches[0]:
    raise SystemExit('Archive checksum mismatch or missing')
print('Verified', asset)
