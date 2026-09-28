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
- [x] T2 Benchmark stats: warmup, percentage trimmed mean, stdev/median/min/max; seq-vs-conc equivalence check with tolerance; tests. Route: delegated.
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

### T2 — Benchmark stats + equivalence (done)
- New `stats.go`: `DurationStats{Mean, TrimmedMean, Median, StdDev, Min, Max, CV}`, `trimmedMean(times []time.Duration, fraction float64) time.Duration` (trims floor(n*fraction) per side, falls back to plain mean when the trim would leave 0 or fewer elements), `computeStats(times, trimFraction) DurationStats` (sample stddev, n-1 denominator; CV = StdDev/Mean).
- New `equivalence.go`: `verifyEquivalence(seq, conc *LinearRegression, tol float64) error` using the existing `maxModelDifference`.
- `benchmark.go` reworked: `BenchmarkResult` now carries `Stats DurationStats` instead of a bare `TrimmedMean` field; `benchmarkSequential`/`benchmarkConcurrent` gained `warmup int, trimFraction float64` params and a `runWarmup` helper that runs+discards `warmup` training runs (with a GC before each) before the measured runs. Old fixed drop-min/drop-max `calculateTrimmedMean` removed in favor of `trimmedMean`/`computeStats`.
- `main.go`: `runQuickComparison` now returns an `error`; after computing sequential/concurrent metrics it calls `verifyEquivalence(sequentialModel, concurrentModel, 1e-9)`, prints OK/FAILED, and returns the error on mismatch. `runPipeline` treats a returned equivalence error as fatal (`os.Exit(1)`) for quick/all modes. `runBenchmarkMode` passes `config.Warmup`/`config.TrimFraction` through and uses `result.Stats.TrimmedMean` for speedup/printing.
- Tests (RED→GREEN):
  - `stats_test.go`: `TestTrimmedMeanBasic`, `TestTrimmedMeanFallsBackToMeanWhenNothingLeft`, `TestTrimmedMeanNoTrim`, `TestTrimmedMeanEmpty`, `TestComputeStatsKnownInput` (known 5-value dataset: Mean=280ns, TrimmedMean=100ns, Median=100ns, Min/Max=100/1000ns, StdDev≈402ns sample stddev, CV≈1.4357 — cross-checked against `math.Sqrt(162000)`), `TestComputeStatsSingleValue`, `TestComputeStatsEmpty`.
  - `equivalence_test.go`: `TestVerifyEquivalenceWithinTolerance`/`ExceedsTolerance`; `buildSyntheticSamples(n, seed)` builds an in-memory `[]Sample` (all categoricals at reference index -1) so training doesn't depend on the missing SPARCS CSV; `TestSequentialConcurrentEquivalenceAcrossWorkerCounts` trains sequential once and concurrent for workers {1,2,4,8} on the same synthetic data and asserts `verifyEquivalence(..., 1e-9)` passes for all; `TestConcurrentTrainingIsDeterministic` trains concurrent twice with the same data/workers and asserts bit-exact equal weights/bias.
  - RED: before `stats.go`/`equivalence.go` existed, `go build ./...` failed with `undefined: trimmedMean / computeStats / verifyEquivalence` (confirmed via `go test -run ... -v`).
  - GREEN: after implementing both files, all new tests passed on first run except `TestComputeStatsKnownInput`'s CV assertion, which failed by a benign quantization artifact (`stats.StdDev` truncates to an `int64` `time.Duration`, so the derived CV loses precision vs. the raw float64 calculation: got 1.4357142857142857, wanted ~1.4374722712498649); widened the test's CV delta from 0.001 to 0.01 to account for that expected truncation — GREEN after that.
- Verification: `gofmt -l .` clean for all touched/added files (`main.go`, `benchmark.go`, `stats.go`, `stats_test.go`, `equivalence.go`, `equivalence_test.go`, `config.go`, `config_test.go`); only the same pre-existing 9 legacy files remain unformatted (one fewer than before T2 — `benchmark.go` is now gofmt-clean since it was rewritten). `go vet ./...` clean. `go build ./...` clean. `go test ./... -count=1` green (all cases, including T1's).
- Gotcha found: an intermediate `git stash -u && git stash pop` (used only to confirm which gofmt failures pre-existed on `main`) round-tripped several already-tracked files through Windows `core.autocrlf`, silently converting `main.go`/`config.go`/`config_test.go`/`benchmark.go` from LF to CRLF and making `gofmt -l` flag them as needing reformatting even though their content was fine. Fixed by stripping `\r` back out; content was unaffected (confirmed via `git diff`). Worth remembering for future sessions on this repo/OS.
- Commit: T2_COMMIT_HASH (filled after commit).
