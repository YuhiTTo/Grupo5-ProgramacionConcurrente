# PC2 — Índice de evidencia y checklist de rúbrica

Este directorio documenta cómo el código de este repositorio cubre cada
punto de la rúbrica de PC2 (`Documentos/CC65_PCs_TP-202620.pdf`). Cada
documento numerado profundiza en un aspecto; esta página solo mapea
punto de rúbrica → archivo(s)/evidencia → estado.

## Checklist de rúbrica

| Punto | Descripción | Archivo(s) / evidencia | Estado |
|---|---|---|---|
| (a) | Algoritmo concurrente inicial + modelo Promela de la lógica de sincronización, libre de condiciones de carrera | [`01-algoritmo-concurrente.md`](01-algoritmo-concurrente.md), [`02-modelo-promela.md`](02-modelo-promela.md), `promela/regression_workers.pml` | listo (verificación de seguridad ya corrida — ver `results/promela/verificacion_spin.txt`; las variantes con LTL y multi-worker nuevas están pendientes de ejecutar) |
| (b) | Go secuencial vs. concurrente con capturas de evidencia | `sequential.go`, `concurrent.go`, `main.go` (`-mode=quick`/`all`), [`06-evidencias.md`](06-evidencias.md) | pendiente: captura (requiere correr con el dataset real) |
| (c) | Explicación del algoritmo concurrente y mecanismos de sincronización (Worker Pools/Pipelines) | [`01-algoritmo-concurrente.md`](01-algoritmo-concurrente.md) | listo |
| (d) | Speedup = T-seq/T-conc y media recortada sobre múltiples corridas, tabla + estadísticas | `stats.go`, `benchmark.go`, `report.go`, [`03-metodologia-benchmark.md`](03-metodologia-benchmark.md) | pendiente: ejecutar con dataset real (código y tests listos; tabla de resultados debe pegarse desde `results/benchmark/speedup_summary.md`) |
| (e) | Análisis de speedup, escalabilidad y trade-offs | [`04-analisis-speedup-escalabilidad.md`](04-analisis-speedup-escalabilidad.md) | pendiente: completar con valores medidos (marco de análisis y fórmulas ya documentados) |
| (f) | Análisis de uso de recursos de cómputo hasta el punto de equilibrio | `resource_profile.go`, `report.go` (`findEquilibrium`/`findEfficiencyDrop`), [`05-recursos-punto-equilibrio.md`](05-recursos-punto-equilibrio.md) | pendiente: ejecutar con dataset real (código y tests listos) |
| (g) | Historial de gitflow | Historial de commits de este repositorio (`git log`), ramas `feature/*` sobre `develop`/`main` | listo (ver `git log --graph --oneline --all`) |

Nota sobre "listo" en (a): la corrida de verificación de seguridad
(`pan -a`, sin propiedades LTL) ya se ejecutó sobre una versión anterior
del modelo (`results/promela/verificacion_spin.txt`, 0 errores). Las
propiedades LTL (`safe_update`, `termination`, `mutex`) y las variantes
`-DNUM_WORKERS=2 -DNUM_JOBS=4` / `-DNUM_WORKERS=4 -DNUM_JOBS=8` son
nuevas (ver `02-modelo-promela.md`) y todavía no se ejecutaron porque
Spin/gcc no están instalados en este entorno de desarrollo.

## Cómo generar la evidencia pendiente

1. Descargar el dataset (`dataset/README.md` y
   `dataset/Link del Dataset Original y Limpio.txt`).
2. Correr `go run . -mode=all -runs 10 -epochs 100` (u otros modos —
   ver el README raíz) para generar `results/benchmark/*`,
   `results/resources/resources.csv` y `results/environment.md`.
3. Capturar pantalla según `06-evidencias.md`.
4. Instalar Spin + gcc (o usar Docker, ver comentarios en
   `promela/run_spin.sh`) y correr `promela/run_spin.sh`/`run_spin.ps1`
   para las variantes y propiedades LTL nuevas.
5. Pegar las tablas generadas (`results/benchmark/speedup_summary.md`)
   en `03-metodologia-benchmark.md` y completar los `TODO(equipo)` de
   `04-analisis-speedup-escalabilidad.md` y
   `05-recursos-punto-equilibrio.md`.

## Documentos

- [`01-algoritmo-concurrente.md`](01-algoritmo-concurrente.md) — algoritmo secuencial vs. concurrente, particionamiento, sincronización.
- [`02-modelo-promela.md`](02-modelo-promela.md) — modelo formal, propiedades, cómo ejecutarlo.
- [`03-metodologia-benchmark.md`](03-metodologia-benchmark.md) — metodología estadística del benchmark.
- [`04-analisis-speedup-escalabilidad.md`](04-analisis-speedup-escalabilidad.md) — marco de análisis de speedup/escalabilidad/trade-offs.
- [`05-recursos-punto-equilibrio.md`](05-recursos-punto-equilibrio.md) — uso de recursos y punto de equilibrio.
- [`06-evidencias.md`](06-evidencias.md) — checklist de capturas de pantalla a tomar.
