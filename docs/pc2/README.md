# PC2 — Índice de evidencia y checklist de rúbrica

Este directorio documenta cómo el código de este repositorio cubre cada
punto de la rúbrica de PC2 (`Documentos/CC65_PCs_TP-202620.pdf`). Cada
documento numerado profundiza en un aspecto; esta página solo mapea
punto de rúbrica → archivo(s)/evidencia → estado.

## Checklist de rúbrica

| Punto | Descripción | Archivo(s) / evidencia | Estado |
|---|---|---|---|
| (a) | Algoritmo concurrente inicial + modelo Promela de la lógica de sincronización, libre de condiciones de carrera | [`01-algoritmo-concurrente.md`](01-algoritmo-concurrente.md), [`02-modelo-promela.md`](02-modelo-promela.md), `promela/regression_workers.pml` | listo (seguridad + LTL verificadas, `errors: 0` — ver `results/promela/`) |
| (b) | Go secuencial vs. concurrente con capturas de evidencia | `sequential.go`, `concurrent.go`, `main.go` (`-mode=quick`/`all`), [`06-evidencias.md`](06-evidencias.md) | listo: valores oficiales en el informe (`Documentos/CC65-PC2-202620-Grupo5.pdf`, Fig. 10); capturas versionadas en el repo son opcionales |
| (c) | Explicación del algoritmo concurrente y mecanismos de sincronización (Worker Pools/Pipelines) | [`01-algoritmo-concurrente.md`](01-algoritmo-concurrente.md) | listo |
| (d) | Speedup = T-seq/T-conc y media recortada sobre múltiples corridas, tabla + estadísticas | `stats.go`, `benchmark.go`, `report.go`, [`03-metodologia-benchmark.md`](03-metodologia-benchmark.md) | listo: tabla oficial completa (speedup + eficiencia); falta desviación estándar/CV por configuración (ver nota en `03-metodologia-benchmark.md`) |
| (e) | Análisis de speedup, escalabilidad y trade-offs | [`04-analisis-speedup-escalabilidad.md`](04-analisis-speedup-escalabilidad.md) | listo: Karp–Flatt calculado, contraste con hardware (6 núcleos físicos) y trade-offs completados |
| (f) | Análisis de uso de recursos de cómputo hasta el punto de equilibrio | `resource_profile.go`, `report.go` (`findEquilibrium`/`findEfficiencyDrop`), [`05-recursos-punto-equilibrio.md`](05-recursos-punto-equilibrio.md) | listo: tabla de recursos oficial, punto de equilibrio recalculado (8 workers) y comparado con la elección del informe (12 workers) |
| (g) | Historial de gitflow | Historial de commits de este repositorio (`git log`), ramas `feature/*` sobre `develop`/`main` | listo (ver `git log --graph --oneline --all`) |

Mejoras sugeridas para el informe (no bloquean el repositorio, pero sí
la nota si no se incorporan): ver
[`07-observaciones-informe.md`](07-observaciones-informe.md).

Nota sobre (a): además de la corrida original de la PC2
(`results/promela/verificacion_spin.txt`, 0 errores), las propiedades
LTL (`safe_update`, `termination`, `mutex`) se verificaron con Spin
6.5.2 en las variantes 2/4, 3/4 y 4/4, todas con `errors: 0`
(`results/promela/w<N>_j<M>_*.txt`, tabla en `02-modelo-promela.md`).

## Estado de la evidencia numérica (03–06)

Los valores de speedup, eficiencia, recursos y el análisis Karp–Flatt
en `03-metodologia-benchmark.md`, `04-analisis-speedup-escalabilidad.md`
y `05-recursos-punto-equilibrio.md` ya están completos, tomados del
informe oficial (`Documentos/CC65-PC2-202620-Grupo5.pdf`). Queda
pendiente únicamente la desviación estándar/CV por configuración (no
reportada en el informe) — ver la nota en `03-metodologia-benchmark.md`
para cómo obtenerla.

## Cómo reproducir/ampliar la evidencia

1. Descargar el dataset (`dataset/README.md` y
   `dataset/Link del Dataset Original y Limpio.txt`).
2. Correr `go run . -mode=all -runs=7 -trim=0.15 -epochs 100` (ver
   `03-metodologia-benchmark.md` para la metodología exacta del
   informe, o el README raíz para otros modos) para generar
   `results/benchmark/*`, `results/resources/resources.csv` y
   `results/environment.md`.
3. Capturar pantalla según `06-evidencias.md` (opcional: el informe ya
   incluye las capturas oficiales).
4. Instalar Spin + gcc (o usar Docker, ver comentarios en
   `promela/run_spin.sh`) y correr `promela/run_spin.sh`/`run_spin.ps1`
   para las variantes y propiedades LTL nuevas.

## Documentos

- [`01-algoritmo-concurrente.md`](01-algoritmo-concurrente.md) — algoritmo secuencial vs. concurrente, particionamiento, sincronización.
- [`02-modelo-promela.md`](02-modelo-promela.md) — modelo formal, propiedades, cómo ejecutarlo.
- [`03-metodologia-benchmark.md`](03-metodologia-benchmark.md) — metodología estadística del benchmark.
- [`04-analisis-speedup-escalabilidad.md`](04-analisis-speedup-escalabilidad.md) — marco de análisis de speedup/escalabilidad/trade-offs.
- [`05-recursos-punto-equilibrio.md`](05-recursos-punto-equilibrio.md) — uso de recursos y punto de equilibrio.
- [`06-evidencias.md`](06-evidencias.md) — checklist de capturas de pantalla a tomar.
- [`07-observaciones-informe.md`](07-observaciones-informe.md) — observaciones y textos listos para pegar en el informe Word (correcciones PC1, equilibrio, Karp–Flatt, riesgos de entrega).
