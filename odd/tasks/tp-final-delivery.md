# TP final delivery

**Objective:** leave the TP report, its evidence and a presentation deck ready
for the final delivery and the video.

**Problem:** commit `7c703bb` (PR #18) states that `gofmt -l` still reports
`preprocessing.go`; that is a Windows CRLF false positive (the committed file is
formatted). PR #18 is only on `develop`. The Word report needs a full
consistency pass (texts, figures, numbers) against the repository evidence, and
there is no slide deck for the presentation recorded in the video.

**Scope:** `docs/tp/03-informe-gaps-ia.md`, the three
`Documentos/CC65-TP-202620-*.docx` and the PDF, a new
`Documentos/CC65-TP-202620-Grupo5-Exposicion.pptx`, `docs/tp/` index links.
Out of scope: section 16 conclusions wording (each member rewrites it) and the
video link placeholder.

**TDD:** not applicable (documentation only; no Go behavior change). Checks:
`gofmt` on committed content, docx `validate.py`, Word COM open/export, page
render review, pptx render review, numbers cross-checked against `docs/` and
`results/`.

## Tasks

- [x] T1 — Fix the GAP-19 reasoning in `docs/tp/03-informe-gaps-ia.md` (three
      places). Route: inline (one mechanical file).
- [x] T2 — Full audit of the Word report against the repository evidence.
      Route: delegated read-only auditor (4+ files).
- [x] T3 — Apply the audit fixes to the three Word files, rebuild TOC, export
      PDF. Route: inline script (proven method).
- [x] T4 — Presentation deck aligned with `docs/tp/05-guion-video.md`, using
      `docs/tp/img/` figures. Route: delegated writer (pptx skill).
- [ ] T5 — Link the deck from `docs/tp/README.md`, commit, PR to develop and
      main.

## Progress

- Branch: `docs/tp-final-delivery` (from `origin/develop`, includes PR #18).
- T2: audit found the trimmed-mean formula with "≤" instead of "+", stale
  section 12 (PR list, Figs. 15/16), unsorted APA list with uncited Holzmann,
  stale GAP-19 wording, orphan headings (headings had keepNext explicitly off)
  and typos.
- T3: fixed via XML scripts (formula, APA order/italics/hanging indent/left
  align, PR list #7–#18 with branch-only bold, Littlepie4 note, Holzmann
  citation, typos, keepNext on 86 headings) plus Word COM for remaining
  italics, TOC and PDF. validate.py PASSED x3; 58 pages; figures 1–20; no
  orphan captions or headings.
- T4: 14-slide deck (`Documentos/CC65-TP-202620-Grupo5-Exposicion.pptx`),
  rendered with LibreOffice and reviewed slide by slide.
- Pending after merge: refresh Figs. 15/16 and the deck's slide 13 with the
  final GitHub history.
