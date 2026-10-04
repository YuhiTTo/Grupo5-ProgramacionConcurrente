# Feature: tp-delivery

## Objective
Leave the TP (week 7) deliverable ready per `Documentos/CC65_PCs_TP-202620.pdf`:
q Spin (deadlock + mutual exclusion), r AI GAP analysis in .md, s conclusions,
v video, gitflow, report starting with PC1+PC2 corrections, Word per student.
Plan: `~/.claude/plans/ayudame-a-creae-un-twinkling-candy.md` (approved 2026-10-03).
User decision: report AND fix important GAPs.

## TDD
Strict TDD enabled (session config). Runner: `go test ./...`.

## Tasks
- [x] T0 — Commit PC2 report fixes (cover code, TOC with 5.2/10.4, regenerated PDF). Route: inline.
- [x] T1 — Spin mutants (race + deadlock) proving the checks detect real errors; docs/tp/01. Route: delegated writer.
- [x] T2 — Structured prompt + AI GAP report (docs/tp/02, 03). Route: delegated analyst.
- [x] T3 — Fix high/medium GAPs with TDD; `go test -race` via Docker. Route: delegated writer.
- [x] T4 — Conclusions draft (docs/tp/04). Route: delegated writer.
- [x] T5 — TP Word report + per-student copies + participation template + PDF. Route: inline scripts.
- [ ] T6 — docs/tp/README, video script, README link, PRs to develop → main.

## Out of scope (team)
Own commits from José/Lucero, recording the video, rewriting conclusions in their words,
individual uploads, participation report percentages.

## Progress / evidence
- T0 cd142a2: cover code fixed; TOC rebuilt via Word COM (field used English style names).
- T1 44870b9: race + deadlock mutants, all caught (errors: 1); run_spin_mutants.sh exit 0 in Docker;
  claims checked against trails (in_update=2; Worker sets updating with result_count=0).
- Participation template b537bab (validate.py PASSED, rendered).
- T2: docs/tp/02 + 03 (19 GAPs: 0 Alta, 4 Media, 15 Baja), analysed commit cd142a2.
- T3 e171945..1142d86 (writer, strict TDD): GAP-01,02,03,04,05,11,12,18,19 fixed; RED evidence per
  behavior change (GAP-12 Close-error path only helper-tested). go vet/test ok, quick OK,
  Docker go test -race ok, missing dataset exits 1 (parent re-ran test + exit code).
- Baseline go test -race (Docker golang:1.27, Go 1.27.1) before fixes: ok.
- Review review-37cd6af1b1bcd522 (T1 slice) reached correction_required, then scope moved because
  T3 committed on the same branch (orchestration error: parallel writer during open review).
  Abandon needs maintainer authorization -> left untouched; fresh slice reviews run in a separate
  git worktree instead.
- T5: Documentos/CC65-TP-202620-{U201714492,U202216120,U202218044}.docx + CC65-TP-202620-Grupo5.pdf
  (52 pages). Built from the revised PC2 report: cover PC2->TP, TP paragraph in summary, new
  13 Correcciones PC2, 14 Spin (+mutants), 15 IA analysis, 16 conclusions, 17 annex
  ([ENLACE DEL VIDEO] placeholder), references renumbered to 18 + Holzmann (2003).
  Word renamed heading style ids to Ttulo1/Ttulo2 after the COM save; script adapted.
  validate.py PASSED; Word TOC update shows 13-18; pages 48-50 rendered and inspected.
