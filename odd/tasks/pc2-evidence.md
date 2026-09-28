# Feature: pc2-evidence

## Objective
Close the PC2 rubric gaps (CC65_PCs_TP-202620.pdf): persisted benchmark evidence, statistics, equivalence tests, equilibrium point, CLI, Promela LTL, and docs.

## Problem / Why
Code is sound but all results go to stdout only; rubric points 3-6 (11/20 pts) lack deliverable evidence.

## Constraints
- Go stdlib only (no third-party libs).
- Dataset (`dataset/SPARCS_2022_clean_go.csv`) is not present locally; real benchmark numbers must be produced by the team.
- Spin and gcc are not installed locally; `go test -race` needs cgo/gcc on Windows.
- TDD: strict (global config), runner `go test ./...`.
- Artifacts in English code; docs in Spanish (project language) neutral register.

## Tasks
- [x] T1 CLI flags (`-mode`, `-runs`, `-epochs`, `-workers`, `-out`), keep `cpu-profile` compat, expose `-mode=clean`. Route: delegated.
- [ ] T2 Benchmark stats: warmup, percentage trimmed mean, stdev/median/min/max; seq-vs-conc equivalence check with tolerance; tests. Route: delegated.
- [ ] T3 Persist results (CSV + Markdown + environment metadata) and resources CSV + equilibrium point; tests. Route: delegated.
- [ ] T4 Promela: LTL properties, variant config, run script, Promela<->Go mapping doc. Route: delegated.
- [ ] T5 Docs `docs/pc2/*` + README update + evidence checklist. Route: delegated.

## Acceptance
- `go vet ./...`, `go build ./...`, `go test ./...` green.
- Running `go run . -mode=benchmark` writes results under `results/`.

## Progress / Evidence
(updated per task)

### T1 — CLI flags (done)
- New `config.go`: `Config` struct + `parseConfig(args []string) (Config, error)` (flag.ContinueOnError-based), `defaultConfig()`, `parseWorkerList`, `joinInts`, `Config.validate()`.
  - Flags: `-mode` (quick|benchmark|resources|cpu-profile|clean|all, default `all`), `-runs` (default 10), `-warmup` (default 1), `-epochs` (default 100, same as before), `-workers` (comma list, default `1,2,4,8,12,16,24,32`), `-trim` (fraction trimmed PER SIDE, default 0.1 = 10% low + 10% high), `-out` (default `results`), `-dataset` (default `dataset/SPARCS_2022_clean_go.csv`).
  - Legacy compat: positional `go run . cpu-profile` still sets `-mode=cpu-profile`; an explicit `-mode=` after it overrides.
  - `-mode=clean` routes to the existing `runCleaning()` (main.go) without touching the regression pipeline.
- `main.go` refactored: `main()` now only parses config and dispatches; the former monolithic `main()` body was split into `runPipeline`, `runQuickComparison`, `runBenchmarkMode`, `runResourcesMode`, gated by `config.Mode` (`all` runs quick+benchmark+resources exactly like the previous unconditional behavior, just parameterized by `config.Epochs`/`config.Runs`/`config.Workers`).
- Tests (`config_test.go`, TDD RED→GREEN):
  - RED: `go test ./... -run TestParseConfig -v` failed to build (`undefined: parseConfig`) before `config.go` existed.
  - GREEN: after implementing `config.go`, all 11 `TestParseConfig*` cases pass (defaults, custom flags, invalid mode, invalid/empty/zero workers, trim out of range, invalid runs/epochs/warmup, legacy `cpu-profile` positional, legacy + trailing flags, `-mode` overriding legacy).
- Verification: `gofmt -l .` still flags only the 10 pre-existing files (`benchmark.go`, `cleaning.go`, `concurrent.go`, `cpu_profile.go`, `dataset.go`, `preprocessing.go`, `regression.go`, `resource_profile.go`, `scaling.go`, `sequential.go`) that were already unformatted on `main` before this branch (confirmed via `git stash` diff) — `main.go`/`config.go`/`config_test.go` are clean. `go vet ./...` clean. `go build ./...` clean. `go test ./... -count=1` green (11/11).
- Commit: T1_COMMIT_HASH (filled after commit).
- Note: dataset CSV absent locally, so `go run .` (mode=all/quick/benchmark/resources) cannot be exercised end-to-end here; `go run . -h` was used to verify flag wiring instead.
