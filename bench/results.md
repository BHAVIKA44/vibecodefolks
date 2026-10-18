# Benchmark results

Linux-6.12.76-linuxkit-aarch64-with-glibc2.41 · aarch64 · go version go1.25.14 linux/arm64 · git version 2.47.3

Release-like binary: 6619320 bytes. Startup samples: 20; analysis samples: 3.

| Fixture | Files / changes | Startup p50 / p95 ms | Discovery p50 ms | Analyze p50 ms | Evidence p50 ms | HTTP changes p50 ms |
| --- | --- | --- | --- | --- | --- | --- |
| small | 101 / 5 | 6.4 / 7.0 | 2.8 | 19.8 | 21.6 | 21.7 |
| medium | 1001 / 30 | 6.5 / 7.2 | 2.9 | 72.6 | 54.7 | 74.9 |
| large | 10001 / 150 | 6.3 / 6.6 | 3.0 | 594.8 | 115.5 | 561.4 |
