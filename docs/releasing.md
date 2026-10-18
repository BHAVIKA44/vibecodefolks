# Publishing a release

Release only reviewed, green changes from `main`. Never overwrite a release or move an existing tag. Check that the intended version does not already exist, and review the changelog and release workflow before tagging.

1. Merge the reviewed pull request after required checks pass. Fetch and inspect the resulting `main` commit.
2. Create an annotated tag such as `git tag -a v0.1.1 -m "VibeCodeFolks v0.1.1" <main-commit>` and push that tag with `git push origin v0.1.1`.
3. The version-tag workflow runs checks, builds and hashes five archives, validates archive contents, and runs matching native smoke tests. Publication requires every job to succeed. Review the Actions run and published release, download all assets and verify `sha256sum -c checksums.txt` (or the platform equivalent).
4. In a disposable environment, test the published installer and verify `vibecodefolks --version` and the local dashboard.

If any step fails, do not bypass protections or replace a published tag. Diagnose and fix on a new branch; use a new version if the original tag must remain immutable. See [benchmark methodology](benchmarks.md) for reproducible performance measurements. Archives are unsigned and checksum integrity relies on the trusted GitHub release channel.
