# Feature: pc2-report-review

## Objective
Review the delivered PC2 report (`Documentos/CC65-PC2-202620-Grupo5.pdf`) and the
repository against the PC2 rubric (`Documentos/CC65_PCs_TP-202620.pdf`), and close
the gaps that can be closed from the repository.

## Problem
- Repository docs (`docs/pc2/03`–`06`) still contain `TODO(equipo)` placeholders
  while the report already has the official measurements.
- New LTL properties (`safe_update`, `termination`, `mutex`) were never run in Spin.
- Report gaps: no explicit "PC1 corrections" section, equilibrium point (12 workers)
  is not supported by its own data (8 workers reaches 96.4 % of max speedup with
  E = 0.588), no dispersion statistics, no Amdahl/Karp–Flatt analysis.
- Repository benchmark defaults (10 runs, 10 % trim) differ from the report
  methodology (7 runs, drop min and max) without a documented reproduction command.

## Scope (authorized: "vuelve a revisar todo ... en caso de que falte o se deba mejorar algo de trabajo o informe hazlo")
- Repository docs and evidence only. The Word/PDF report itself is not in the repo
  as an editable source; report improvements are delivered as ready-to-paste text.
- No fabricated measurements: only numbers from the report or from real runs.

## Constraints
- Spanish, neutral/professional register for `docs/pc2/*` (existing language).
- Conventional commits, no AI attribution (user CLAUDE.md).
- TDD: strict mode configured; this feature changes docs/evidence and one
  PowerShell script (no Go behavior change), so `go test ./...` is the regression check.

## Tasks
- [x] R1 — Run Spin (variants 3/4, 2/4, 4/4 — 4/8 aborted, see evidence: safety + 3 LTL) via Docker, commit
      `results/promela/*`, update `docs/pc2/02-modelo-promela.md`; make
      `run_spin.ps1` fail on non-zero native exit codes. Route: inline (commands).
- [x] R2 — Replace `TODO(equipo)` in `docs/pc2/03`–`06` with the report's official
      measurements (Ryzen 5 9600X), Karp–Flatt table, equilibrium criteria and the
      reproduction command `-runs=7 -trim=0.15`. Route: delegated writer (2+ files).
- [x] R3 — Add `docs/pc2/07-observaciones-informe.md` with ready-to-paste report
      additions (PC1 corrections, equilibrium revision, Karp–Flatt, Spin LTL,
      delivery checklist). Route: same delegated writer.

## Acceptance criteria
- No `TODO(equipo)` left in `docs/pc2`.
- `results/promela/` holds one safety file and three LTL files per variant, all with
  `errors: 0`, or the failure is documented honestly.
- Every number in the docs traces to the report or to a committed result file.
- `go test ./...` passes.

## Checks
- `go test ./...`, `go vet ./...`
- `grep -rn "TODO(equipo)" docs` returns nothing.

## Delivery
- Branch `feature/pc2-report-review` from `main` (518527b), PR to `develop`.
- Forecast: ~400 authored lines (mostly docs). Strategy: `ask-on-risk`; single PR expected.

## Progress / evidence
- 2026-09-28: report and rubric reviewed; all report arithmetic re-computed and
  consistent (speedup, efficiency, 1.21 %, +30 % allocs).
- R1 done (inline): Spin 6.5.2 via Docker debian:stable-slim; 2/4, 3/4, 4/4 safety + 3 LTL
  all `errors: 0`. 4/8 aborted at >34M states / 3.2 GB (documented). run_spin.ps1 checks
  $LASTEXITCODE (PowerShell parser: 0 errors). Commit fd8d72f.
- R2/R3 done (delegated writer, trigger: 2+ non-trivial files). Karp–Flatt e: p2 0.0176,
  p4 0.0308, p8 0.1000, p12 0.1353, p16 0.1518, p24 0.1712, p32 0.1799 (parent re-checked
  p16/p24/p32). `grep TODO(equipo) docs` empty; `go vet ./...` clean; `go test ./...` ok.

- Review review-9ebd7673f81d6276 (4 lenses, consent granted): approved, acknowledged, authority
  burned. Follow-up for advisory findings R2-1, R2-2, R3-1, R4-002: scripts now fail when pan
  output lacks `errors: 0` (tested: real file ok, fake `errors: 3` rejected; Docker 2/4 run exit 0).
  R2-3 dismissed (flags already default). R1-001/R1-002/R3-2/R4-001/R4-003 left as informational.

- Review review-7b40d84ace7f022c (a4e9b6a, 4 lenses, consent granted): approved, acknowledged,
  authority burned. Advisory warnings left informational.
- [x] R4 (added on user request "modifica el informe"): Word report edited as a new copy
  `Documentos/CC65-PC2-202620-[Código de alumno]-revisado.docx` (original kept byte-identical):
  new 5.2 PC1 corrections, LTL table at end of 6.4, new 10.4 Karp–Flatt, 8 vs 12 workers
  complement in 11.3, branches/PRs in 12, Karp & Flatt (1990) reference; updateFields=true so
  Word offers to refresh the TOC. Checks: docx skill validate.py PASSED; Word COM export OK
  (46 pages); pages 14/22/40/44/46 rendered and inspected.

## Next step
- Push branch and open PR to `develop` (user decision).
- Team: paste docs/pc2/07 blocks into the Word report; upload one Word per student.
