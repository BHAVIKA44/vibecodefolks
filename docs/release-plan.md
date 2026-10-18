# Future release plan

No release artifacts are published by this branch. After quality review, a future release workflow can build standalone executables for macOS (arm64 and amd64), Linux (arm64 and amd64), and Windows (amd64), verify their checksums, and attach versioned archives and checksums to a GitHub Release. CI should run before publishing. Installation and release packaging are separate future work; cross-compiling alone does not verify execution on each operating system. Signing and platform-specific distribution should be evaluated before release.
