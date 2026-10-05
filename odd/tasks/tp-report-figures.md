# TP report figures

**Objective:** bring the TP Word report's figures up to date and add the
missing evidence figures before delivery.

**Problem:** the GitHub history screenshot stops at PR #3 and has no caption.
Figure 5 shows the pre-PC2 Spin model. Sections 14 (Spin) and 15 (AI) have no
figures, Figure 10 is a quick run without clarification, and efficiency,
Karp–Flatt and memory have tables but no charts.

**Scope:** `docs/tp/img/` (charts, terminal captures, generator script) and the
three `Documentos/CC65-TP-202620-*.docx` plus `CC65-TP-202620-Grupo5.pdf`.
Out of scope: section 16 (conclusions, rewritten by each member) and the
video link placeholder.

**TDD:** not applicable (documentation and generated images; no Go behavior
changes). Checks: docx `validate.py`, Word COM open/export, page render review,
chart values cross-checked against `docs/pc2/04`, `docs/pc2/05`, `docs/tp/03`.

## Tasks

- [x] T1 — Charts (efficiency, Karp–Flatt, memory, GAP status) with
      `docs/tp/img/make_charts.py`. Route: delegated (writer + generation).
- [x] T2 — Real terminal captures (Spin correct model, mutants, `go test -race`)
      from Docker runs, rendered to PNG with the raw `.txt` kept. Route: delegated
      with T1.
- [x] T3 — GitHub commit history screenshot of `main` (headless browser).
      Route: inline.
- [x] T4 — Insert figures into the three Word files, renumber figures, update
      sections 6.4, 9, 10, 11.3, 12, 14, 15, rebuild TOC, export PDF.
      Route: inline (script already proven in tp-delivery).
- [ ] T5 — Commit, PR to develop and main.

## Progress

- Branch: `docs/tp-report-figures`.
- T1/T2: charts and real Docker captures in `docs/tp/img/` (Spin 3/4 errors: 0;
  mutants errors: 1 each; `go test -race -v` 94 PASS, 0 DATA RACE). Delegated.
- T3: headless Chrome capture of `main` commits plus `git shortlog -sn` render.
  GitHub Contributors attributes only 1 commit to YuhiTTo (teammate emails not
  linked to the account), so it was not used as a figure.
- T4: 9 images per docx, figures 1–20 consecutive, captions keepNext, 23 level-2
  headings (6.1–11.3) made bold, TOC updated, 58-page PDF; validate.py PASSED x3.
