# Feature: dataset-download

## Objective
Anyone who clones the repo can obtain the SPARCS 2022 datasets automatically and safely, without committing them to GitHub.

## Problem / Why
Datasets (clean 244,031,060 B; raw 947,122,735 B) live on Google Drive and are gitignored; setup was manual and unverified.

## Constraints
- Go stdlib only; cross-platform (`go run . -mode=download`).
- Integrity: pinned SHA-256 + exact size; HTTPS only; allowed hosts only; reject HTML (Drive warning pages); write to `.part` then atomic rename; never leave corrupt files.
- Files stay gitignored (`dataset/*.csv`).
- TDD strict, runner `go test ./...` (tests use httptest TLS server, no network).

## Tasks
- [x] D1 Secure downloader (`dataset_download.go`) + tests. Route: delegated (2+ non-trivial files).
- [x] D2 CLI wiring: `-mode=download` (+ which dataset), auto-download clean dataset when missing for data modes, `-no-download` opt-out; tests. Route: delegated.
- [x] D3 Docs: dataset/README.md, README.md, `dataset/SHA256SUMS` for manual verification. Route: delegated.

## Acceptance
- `go vet`, `go build`, `go test ./...` green.
- Real `go run . -mode=download` produces a file whose SHA-256 matches the pinned value; re-running skips the download.

## Progress / Evidence

### D1 — Secure downloader (`dataset_download.go`)
- TDD: wrote `dataset_download_test.go` first (13 test funcs, table-driven where applicable) covering happy path, hash mismatch, truncated body, Content-Length mismatch, HTML rejection, non-200, http-scheme rejection, disallowed-redirect rejection, skip-when-valid, replace-when-corrupt, context cancellation, `verifyFile`, `hostAllowed`, and the `datasetSpecs` registry.
- RED: `go test ./... -run TestEnsureDataset` failed to compile (`undefined: DatasetSpec`, `undefined: ensureDataset`) before implementation existed.
- Implemented `DatasetSpec`, `datasetSpecs` registry (clean/raw, built from the pinned Drive IDs/sizes/hashes), `ensureDataset`, `verifyFile`, `validateDatasetURL` (https-only + host allowlist, parameterized so tests use an httptest TLS server instead of the real network), `streamToFile` (`.part` + atomic `os.Rename`, sha256 computed while streaming, cleanup on any error), and a bucketed (non-time-based, deterministic) progress writer.
- GREEN: `go test ./... -run "TestEnsureDataset|TestVerifyFile|TestHostAllowed|TestDatasetSpecsRegistry" -v` — all 13 tests + subtests PASS.
- Full suite: `go vet ./...` clean, `go build ./...` clean, `go test ./... -count=1` PASS (no regressions).
- Note: `gofmt -l` flags several pre-existing files (cleaning.go, concurrent.go, dataset.go, preprocessing.go, regression.go, scaling.go, sequential.go) due to CRLF line endings from `core.autocrlf`; unrelated to this change, left untouched. New files (`dataset_download.go`, `dataset_download_test.go`) are gofmt-clean.
- Commit: `be115f6`

### D2 — CLI wiring
- TDD: added flag-parsing tests to `config_test.go` (`TestParseConfigDownloadDefaults`, `TestParseConfigModeDownload`, `TestParseConfigDownloadAllAndNoDownload`, `TestParseConfigInvalidDownloadTarget`) and wiring tests in `main_test.go` (`TestEnsureNamedDatasetAvailable_*`, `TestEnsureCleanDatasetAvailable_*`, `TestDownloadTargets`) before implementing.
- RED: editor diagnostics confirmed `config.DownloadTarget`/`config.NoDownload` undefined before the `Config` struct was extended.
- Implemented: `-mode=download` (new valid mode), `-download=clean|raw|all` (default `clean`), `-no-download` bool flag; `Config.validate()` rejects an invalid `-download` value. `main()` routes `-mode=download` to `runDownloadMode` (real network, not exercised by tests) and `-mode=clean` / all other modes through `ensureRawDatasetAvailable` / `ensureCleanDatasetAvailable` before running. `ensureCleanDatasetAvailable` only auto-downloads when `-dataset` is exactly the default clean path; a custom `-dataset` that is missing errors with guidance instead of downloading to an arbitrary path. `-no-download` yields an actionable error pointing at `go run . -mode=download`.
- GREEN: `go test ./... -run "TestParseConfigDownload|TestParseConfigModeDownload|TestParseConfigInvalidDownloadTarget|TestEnsureNamedDatasetAvailable|TestEnsureCleanDatasetAvailable|TestDownloadTargets" -v` all PASS; full suite `go test ./... -count=1` PASS (68+ tests, no regressions), `go vet ./...` clean, `go build ./...` clean, `gofmt -l` clean on touched files.
- Verified manually: `go run . -h` lists the new flags; `go run . cpu-profile -no-download` (legacy positional) still maps to `-mode=cpu-profile` and now correctly reports the actionable no-download error instead of a raw CSV-read failure.
- Incident (self-caught, corrected): an earlier manual smoke check (`go run . cpu-profile -epochs=1 | head -5`) triggered the real auto-download path since no dataset file existed locally and `-no-download` wasn't passed; the ~244 MB file was fully fetched and verified (hash matched) before being deleted immediately, since the real end-to-end download was explicitly reserved for the parent to run. All later manual checks used `-no-download` to stay network-free. No commit or artifact includes that file; `dataset/*.csv` stays gitignored.
- Commit: `1365a67`

### D3 — Docs
- `dataset/SHA256SUMS`: two lines, standard `sha256sum -c` format (`<hash>  <filename>`, filenames relative to `dataset/`), for the pinned clean and raw hashes. Confirmed NOT gitignored: `git check-ignore -v dataset/SHA256SUMS` exited 1 (not ignored) — the `dataset/*.csv` rule only matches `.csv`.
- `dataset/README.md`: added an "automatic download" section (`go run . -mode=download`, `-download=raw|all`, auto-download-on-missing behavior, `-no-download`), what is verified (HTTPS + host allowlist, non-200/HTML rejection, size, SHA-256, atomic `.part` + rename), the manual alternative (Drive links, exact expected filenames), manual verification commands for Linux/macOS (`sha256sum -c` / `shasum -a 256 -c`) and Windows PowerShell (`Get-FileHash -Algorithm SHA256`), why `.csv` files are never committed, and the procedure to update the pinned hash/size if the Drive file is ever replaced.
- Root `README.md`: Dataset section now leads with `go run . -mode=download` (+ `-download=raw|all`, `-no-download`) and links to `dataset/README.md` for detail; flags table gained `-download` and `-no-download`; Modos list gained `download`.
- No code changes in this task; full suite re-verified after the doc edits: `gofmt -l` clean (touched files only — pre-existing CRLF-flagged files from D1 untouched), `go vet ./...` clean, `go build ./...` clean, `go test ./... -count=1` PASS.
- Commit: (recorded after commit below)
