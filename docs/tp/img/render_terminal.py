"""Render a text log as a dark terminal screenshot.

Usage:
  python render_terminal.py LOG.txt OUT.png "command shown after $" \
      [--keep REGEX]... [--lines A-B]...

Only real lines of LOG.txt are shown, in original order. Without --keep/--lines
the whole log is rendered. With them, a line is kept if it matches any REGEX or
falls in any 1-based inclusive range A-B; omitted runs become a single "..." line.
"""
import argparse
import re
from PIL import Image, ImageDraw, ImageFont

BG, FG, DIM = (30, 30, 30), (212, 212, 212), (128, 128, 128)
FONTS = ["C:/Windows/Fonts/consola.ttf", "C:/Windows/Fonts/DejaVuSansMono.ttf",
         "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"]


def load_font(size):
    for path in FONTS:
        try:
            return ImageFont.truetype(path, size)
        except OSError:
            continue
    raise SystemExit("no monospace font found")


def select(lines, patterns, ranges):
    if not patterns and not ranges:
        return lines
    rx = [re.compile(p) for p in patterns]
    out, skipped = [], False
    for i, line in enumerate(lines, 1):
        keep = any(r.search(line) for r in rx) or any(a <= i <= b for a, b in ranges)
        if keep:
            if skipped and out:
                out.append("…")
            out.append(line)
            skipped = False
        else:
            skipped = True
    return out


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("log")
    ap.add_argument("out")
    ap.add_argument("command")
    ap.add_argument("--keep", action="append", default=[])
    ap.add_argument("--lines", action="append", default=[])
    ap.add_argument("--size", type=int, default=16)
    a = ap.parse_args()
    ranges = [tuple(int(x) for x in r.split("-")) for r in a.lines]
    text = open(a.log, encoding="utf-8", errors="replace").read().replace("\t", "    ")
    lines = select([l.rstrip() for l in text.splitlines()], a.keep, ranges)
    lines = ["$ " + a.command] + lines
    font = load_font(a.size)
    asc, desc = font.getmetrics()
    lh = asc + desc + 4
    cw = font.getlength("M")
    pad = 16
    width = int(pad * 2 + cw * max(len(l) for l in lines))
    img = Image.new("RGB", (width, pad * 2 + lh * len(lines)), BG)
    d = ImageDraw.Draw(img)
    for i, l in enumerate(lines):
        color = DIM if l == "…" else FG
        d.text((pad, pad + i * lh), l, font=font, fill=color)
    img.save(a.out)
    print(a.out, img.size)


if __name__ == "__main__":
    main()
