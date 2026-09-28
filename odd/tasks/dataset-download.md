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
- [ ] D1 Secure downloader (`dataset_download.go`) + tests. Route: delegated (2+ non-trivial files).
- [ ] D2 CLI wiring: `-mode=download` (+ which dataset), auto-download clean dataset when missing for data modes, `-no-download` opt-out; tests. Route: delegated.
- [ ] D3 Docs: dataset/README.md, README.md, `dataset/SHA256SUMS` for manual verification. Route: delegated.

## Acceptance
- `go vet`, `go build`, `go test ./...` green.
- Real `go run . -mode=download` produces a file whose SHA-256 matches the pinned value; re-running skips the download.

## Progress / Evidence
(updated per task)
