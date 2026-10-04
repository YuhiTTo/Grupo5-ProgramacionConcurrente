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
- [ ] T0 — Commit PC2 report fixes (cover code, TOC with 5.2/10.4, regenerated PDF). Route: inline.
- [ ] T1 — Spin mutants (race + deadlock) proving the checks detect real errors; docs/tp/01. Route: delegated writer.
- [ ] T2 — Structured prompt + AI GAP report (docs/tp/02, 03). Route: delegated analyst.
- [ ] T3 — Fix high/medium GAPs with TDD; `go test -race` via Docker. Route: delegated writer.
- [ ] T4 — Conclusions draft (docs/tp/04). Route: delegated writer.
- [ ] T5 — TP Word report + per-student copies + participation template + PDF. Route: inline scripts.
- [ ] T6 — docs/tp/README, video script, README link, PRs to develop → main.

## Out of scope (team)
Own commits from José/Lucero, recording the video, rewriting conclusions in their words,
individual uploads, participation report percentages.

## Progress / evidence
