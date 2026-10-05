"""Generate the report charts (Spanish labels). Run: python docs/tp/img/make_charts.py

Data sources (hardcoded below on purpose, so the script is reproducible):
  - docs/pc2/04-analisis-speedup-escalabilidad.md  (speedup, Karp-Flatt)
  - docs/pc2/05-recursos-punto-equilibrio.md       (mallocs per workers)
  - docs/tp/03-informe-gaps-ia.md                  (GAP table, verified at runtime)
"""
import re
import sys
from pathlib import Path

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]

WORKERS = [1, 2, 4, 8, 12, 16, 24, 32]
SPEEDUP = [1.0272, 1.9655, 3.6615, 4.7050, 4.8224, 4.8817, 4.8603, 4.8668]
KARP_FLATT = {2: 0.0176, 4: 0.0308, 8: 0.1000, 12: 0.1353, 16: 0.1518, 24: 0.1712, 32: 0.1799}
MALLOCS_SEQ = 108
MALLOCS = [1109, 1609, 2609, 4610, 6625, 8633, 12622, 16665]


def save(fig, name):
    fig.tight_layout()
    fig.savefig(HERE / name, dpi=150, facecolor="white")
    plt.close(fig)


def line_labels(ax, xs, ys, fmt, **kw):
    for x, y in zip(xs, ys):
        ax.annotate(fmt.format(y), (x, y), textcoords="offset points",
                    xytext=(0, 8), ha="center", fontsize=8, **kw)


def eficiencia():
    eff = [s / p for s, p in zip(SPEEDUP, WORKERS)]
    fig, ax = plt.subplots(figsize=(8, 5))
    ax.plot(WORKERS, eff, marker="o", color="tab:blue", label="Eficiencia medida")
    line_labels(ax, WORKERS, eff, "{:.2f}")
    ax.set_xlim(0, 33)
    ax.axvline(6, linestyle="--", color="tab:red", alpha=0.7, label="6 núcleos físicos")
    ax.axhline(1.0, linestyle="--", color="gray", alpha=0.8, label="eficiencia ideal")
    ax.set_xticks(WORKERS)
    ax.set_ylim(0, 1.2)
    ax.set_xlabel("Cantidad de workers")
    ax.set_ylabel("Eficiencia (speedup / p)")
    ax.set_title("Eficiencia según la cantidad de workers", fontsize=12)
    ax.grid(alpha=0.3)
    ax.legend(loc="upper right")
    save(fig, "eficiencia.png")


def karp_flatt():
    xs = list(KARP_FLATT)
    ys = list(KARP_FLATT.values())
    fig, ax = plt.subplots(figsize=(8, 5))
    ax.plot(xs, ys, marker="o", color="tab:blue")
    line_labels(ax, xs, ys, "{:.4f}")
    ax.set_xticks(WORKERS)
    ax.set_ylim(0, 0.21)
    ax.set_xlabel("Cantidad de workers")
    ax.set_ylabel("e (Karp–Flatt)")
    ax.set_title("Fracción secuencial experimental (Karp–Flatt)", fontsize=12)
    ax.grid(alpha=0.3)
    save(fig, "karp_flatt.png")


def memoria():
    fig, ax = plt.subplots(figsize=(8, 5))
    ax.plot(WORKERS, MALLOCS, marker="o", color="tab:blue", label="Asignaciones (mallocs)")
    ax.axhline(MALLOCS_SEQ, linestyle=":", color="tab:blue", alpha=0.6)
    ax.annotate(f"secuencial: {MALLOCS_SEQ}", (32, MALLOCS_SEQ), textcoords="offset points",
                xytext=(0, 8), ha="right", fontsize=8, color="tab:blue")
    for x, y in zip(WORKERS, MALLOCS):
        low = x <= 2
        ax.annotate(f"{y:,}", (x, y), textcoords="offset points",
                    xytext=((-8, -3) if x == 1 else (8, -14)) if low else (-4, 8),
                    ha=("right" if x == 1 else "left") if low else "right",
                    fontsize=8, color="tab:blue")
    ax.set_xticks(WORKERS)
    ax.set_ylim(0, 19000)
    ax.set_xlabel("Cantidad de workers")
    ax.set_ylabel("Asignaciones de memoria (mallocs)", color="tab:blue")
    ax.set_title("Asignaciones de memoria según la cantidad de workers", fontsize=12)
    ax.grid(alpha=0.3)
    ax.axvspan(7.5, 12.5, color="gold", alpha=0.2)
    ax.text(10, 17600, "punto de equilibrio\n(8 eficiencia / 12 práctico)",
            ha="center", va="center", fontsize=8)
    ax2 = ax.twinx()
    ax2.plot(WORKERS, SPEEDUP, marker="s", color="tab:orange", label="Speedup")
    for x, y in zip(WORKERS, SPEEDUP):
        ax2.annotate(f"{y:.2f}", (x, y), textcoords="offset points",
                     xytext=(0, -14) if x == 32 else (0, 8), ha="center", fontsize=8,
                     color="tab:orange")
    ax2.set_ylim(0, 6)
    ax2.set_ylabel("Speedup", color="tab:orange")
    h1, l1 = ax.get_legend_handles_labels()
    h2, l2 = ax2.get_legend_handles_labels()
    ax.legend(h1 + h2, l1 + l2, loc="center right", fontsize=8)
    save(fig, "memoria.png")


def gaps_estado():
    text = (ROOT / "docs/tp/03-informe-gaps-ia.md").read_text(encoding="utf-8")
    counts = {}
    for m in re.finditer(r"^\| GAP-(\d+) \| [^|]+\| (Alta|Media|Baja) \|.*\| (Corregido|Parcial|Pendiente)[^|]*\|\s*$",
                         text, re.M):
        key = (m.group(2), m.group(3))
        counts[key] = counts.get(key, 0) + 1
    expected = {("Media", "Corregido"): 4, ("Baja", "Corregido"): 3,
                ("Baja", "Parcial"): 2, ("Baja", "Pendiente"): 10}
    if counts != expected:
        sys.exit(f"GAP counts differ from expected: {counts} != {expected}")
    states = [("Corregido", "tab:green"), ("Parcial", "orange"), ("Pendiente", "tab:gray")]
    sev = ["Alta", "Media", "Baja"]
    fig, ax = plt.subplots(figsize=(8, 5))
    left = [0, 0, 0]
    for st, color in states:
        vals = [counts.get((s, st), 0) for s in sev]
        ax.barh(sev, vals, left=left, color=color, label=st, edgecolor="white")
        for i, v in enumerate(vals):
            if v:
                ax.text(left[i] + v / 2, i, str(v), ha="center", va="center",
                        fontsize=10, color="white", fontweight="bold")
        left = [l + v for l, v in zip(left, vals)]
    for i, total in enumerate(left):
        ax.text(total + 0.3, i, f"total: {total}", va="center", fontsize=9)
    ax.set_xlim(0, 18)
    ax.set_xlabel("Cantidad de GAPs")
    ax.set_ylabel("Severidad")
    ax.set_title("GAPs detectados por la IA según severidad y estado", fontsize=12)
    ax.grid(axis="x", alpha=0.3)
    ax.legend(loc="lower right")
    save(fig, "gaps_estado.png")


if __name__ == "__main__":
    eficiencia()
    karp_flatt()
    memoria()
    gaps_estado()
