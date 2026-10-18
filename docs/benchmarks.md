# Performance benchmarks

Measured in a Linux ARM64 Docker Desktop VM on this Mac (8 virtual CPUs, approximately 7.75 GiB assigned memory). The host CPU model was unavailable to this sandbox. Other operating systems and architectures were **not benchmarked**. These results are warm repeated measurements, not controlled cold starts. The benchmark and release build ran concurrently, which may affect timings.

Toolchain: go version go1.25.14 linux/arm64; git version 2.47.3; Linux-6.12.76-linuxkit-aarch64-with-glibc2.41.

## Method

The runner builds a stripped, CGO-disabled binary, generates disposable repositories in its own temporary directory, and removes them afterwards. Fixtures contain mixed extensions, directories, ignored content, staged and unstaged edits, and one untracked file. A separate one-file last commit exercises a clean tree. No fixture code is executed and no AI provider is called. Startup is process launch through the dashboard HTTP 200 response; it includes process startup and the HTTP request, but excludes the separate Git-analysis request. The HTTP endpoint timings include response transfer. Direct Go probes measure repository discovery, Analyze, and Collect separately; Collect can do its own prerequisite work.

Each startup has 20 samples; analysis and endpoint timings have 3. p95 uses nearest rank. CPU scheduling and warm caches affect results. The dashboard response time and process-to-HTTP-ready time are the same end-to-end measurement here, not two independent intervals.

## Startup (milliseconds)

| Fixture (tracked / changed) | Samples | min | p50 | p95 | max |
| --- | ---: | ---: | ---: | ---: | ---: |
| small (101 / 5) | 20 | 6.23 | 6.39 | 7.05 | 19.34 |
| medium (1001 / 30) | 20 | 6.20 | 6.48 | 7.18 | 7.90 |
| large (10001 / 150) | 20 | 6.15 | 6.27 | 6.65 | 7.53 |

## Git analysis (milliseconds, p50 of 3)

| Fixture | Discover | Analyze | Collect evidence | HTTP /api/changes | Clean tree Analyze (single sample) |
| --- | ---: | ---: | ---: | ---: | ---: |
| small | 2.77 | 19.80 | 21.59 | 21.70 | 12.41 |
| medium | 2.86 | 72.57 | 54.71 | 74.86 | 13.59 |
| large | 2.96 | 594.78 | 115.53 | 561.36 | 28.13 |

## Memory

RSS is the Go server process only, sampled from Linux `/proc/<pid>/status`; browser, Git child processes, and Docker VM memory are excluded. Idle is the median of three samples; peaks are the highest observed across three runs sampled approximately every 50 ms, including the idle sample. Short-lived peaks can be missed and are lower bounds. Go allocation figures are cumulative bytes allocated during each direct Analyze call, not RSS or retained heap.

| Fixture | Idle RSS KiB | Changes peak observed KiB | Evidence peak observed KiB | Analyze allocated bytes / allocations (median) |
| --- | ---: | ---: | ---: | ---: |
| small | 10168 | 12084 | 12084 | 1060048 / 2700 |
| medium | 11180 | 14496 | 15312 | 3261392 / 8413 |
| large | 10040 | 15384 | 15440 | 13357048 / 35514 |

## Release executable sizes

| Target | Bytes (uncompressed) |
| --- | ---: |
| macOS arm64 | 6,729,506 |
| macOS amd64 | 7,185,360 |
| Linux arm64 | 6,619,320 |
| Linux amd64 | 7,078,072 |
| Windows amd64 | 7,266,816 |

These are the locally built v0.1.0 cross-compiled executables; sizes can vary with Go toolchain versions. Analyze grew from about 20 ms for 5 changes to 595 ms for 150 changes. Analysis uses multiple Git operations and allocates more for larger diffs; these small synthetic samples do not isolate a safe optimization, so no performance rewrite was made. Large or complex real diffs may behave differently. Provider inference, browser memory, cold caches, and other architectures were not measured. More samples and real-world repositories are needed before drawing broad conclusions.

## Reproduce

Install Git, Go, Python 3, and on macOS a working `ps` command; Python is only a benchmark development dependency. On Linux, RSS comes from `/proc`. From the repository root:

```sh
python3 bench/run.py --iterations 20 --analysis-iterations 3
```

Results are written to `bench/results.json` and `bench/results.md`. The raw file includes min, median, p95, max, allocation counts, and clean-tree samples. If RSS measurement is unavailable, its entries are null; never interpret Go allocation totals as total process memory. The runner only deletes its own temporary workspace.
