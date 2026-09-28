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
- [ ] D2 CLI wiring: `-mode=download` (+ which dataset), auto-download clean dataset when missing for data modes, `-no-download` opt-out; tests. Route: delegated.
- [ ] D3 Docs: dataset/README.md, README.md, `dataset/SHA256SUMS` for manual verification. Route: delegated.

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
- Commit: (recorded after commit below)
