# 03 — Metodología del benchmark

## Diseño de las corridas

- **Runs**: número de mediciones oficiales por configuración
  (`-runs`, default 10, `config.go:14-19,79-83`).
- **Warmup**: corridas de entrenamiento descartadas ANTES de medir
  (`-warmup`, default 1, `runWarmup`, `benchmark.go:18-26`), para
  reducir ruido de arranque en frío (primer GC, caches del runtime,
  etc.). No se incluyen en las estadísticas ni en `Times`
  (`TestBenchmarkWarmupRunsAreExcluded`, `stats_test.go`).
- **Configuraciones de workers**: lista configurable con `-workers`
  (default `1,2,4,8,12,16,24,32`, `config.go:12`); el secuencial
  siempre se mide una vez y se usa como referencia (`Workers: 0`).
- **Controles**:
  - `runtime.GC()` antes de cada corrida individual
    (`benchmark.go:63`, `:133`, y en `runWarmup`, `:23`) para no
    arrastrar basura de la corrida anterior.
  - Mismos datos de entrenamiento (`trainData`) para secuencial y
    todas las configuraciones concurrentes dentro de una misma
    invocación del CLI.
  - Split train/test con semilla fija (`42`, `main.go:118`) para que
    las corridas sean reproducibles.

## Media recortada (trimmed mean)

`trimmedMean(times, fraction)` (`stats.go:27-43`) ordena las
mediciones y descarta `floor(n * fraction)` valores en **cada
extremo** (inferior y superior) antes de promediar el resto:

```
fraction = 0.1  →  recorta 10% inferior + 10% superior (20% del total)
```

Si el recorte dejaría 0 o menos elementos, se usa la media simple
sobre todos los valores (fallback, `stats.go:36-38`). El objetivo es
reducir la influencia de outliers (p. ej. una corrida interrumpida por
el scheduler del SO o un GC largo) sin descartar tantos datos como
para perder representatividad. `fraction` es configurable con `-trim`
(default `0.1`, rango válido `[0, 0.5)`, `config.go:18`, `:209-211`).

## Estadísticas reportadas

`computeStats(times, trimFraction)` (`stats.go:47-73`) calcula, para
cada configuración:

| Estadística | Fórmula / definición | Campo |
|---|---|---|
| Media | Σt / n | `Mean` |
| Media recortada | ver arriba | `TrimmedMean` |
| Mediana | valor central (o promedio de los dos centrales si n es par) | `Median` |
| Desviación estándar (muestral) | `sqrt(Σ(t - media)² / (n-1))`, n-1 en el denominador | `StdDev` |
| Mínimo / Máximo | — | `Min` / `Max` |
| CV (coef. de variación) | `StdDev / Mean`, calculado en punto flotante (ns) ANTES de truncar `StdDev` a `time.Duration`, para no perder precisión por cuantización a nanosegundos enteros | `CV` |

## Speedup y eficiencia

Calculados en `main.go:529-535` (modo `benchmark`) usando la media
recortada como estimador central:

```
Speedup(workers)    = TrimmedMean(secuencial) / TrimmedMean(workers)
Efficiency(workers) = Speedup(workers) / workers
```

Un `Speedup` de 1.0 significa "igual de rápido que el secuencial";
`Efficiency` de 1.0 significa escalabilidad lineal perfecta para esa
cantidad de workers.

## Cómo reproducir

```bash
go run . -mode=all -runs 10 -epochs 100 -warmup 1 -trim 0.1 \
  -workers 1,2,4,8,12,16,24,32 -out results -dataset dataset/SPARCS_2022_clean_go.csv
```

Modos relevantes (ver README raíz para la lista completa de flags):

- `-mode=quick`: comparación única secuencial vs. concurrente (todas
  las CPUs lógicas) + verificación de equivalencia (`1e-9`).
- `-mode=benchmark`: el benchmark formal descrito aquí.
- `-mode=all` (default): corre `quick` + `benchmark` + `resources`.

## Archivos de salida

Generados por `persistBenchmarkResults` (`main.go:573-634`) vía
`report.go`:

- `results/benchmark/benchmark_runs.csv` — una fila por corrida
  individual (`mode, workers, run, duration_ms`).
- `results/benchmark/speedup_summary.csv` y `.md` — una fila por
  configuración, con todas las estadísticas + speedup + eficiencia.
- `results/benchmark/equilibrium.md` — punto de equilibrio y punto de
  caída de eficiencia (ver `05-recursos-punto-equilibrio.md`).
- `results/environment.md` — metadatos de entorno (versión de Go,
  SO/arquitectura, NumCPU, GOMAXPROCS, timestamp, configuración de la
  corrida).

## Resultados

Valores oficiales tomados del informe entregado
(`Documentos/CC65-PC2-202620-Grupo5.pdf`), corrida en un AMD Ryzen 5
9600X (6 núcleos físicos / 12 lógicos, 15.6 GB RAM, Windows 11 Pro
64-bit, Go 1.27.0), dataset completo (2,103,432 filas; 1,682,537 de
entrenamiento — 79.99 % —, 420,895 de prueba; 49 features), 100
épocas, `lr=0.01`, **7 corridas por configuración** con media
recortada quitando el mínimo y el máximo (ver nota de reproducción más
abajo, el default del repositorio usa 10 corridas y recorte de 10 %):

| mode | workers | trimmed_mean_ms | speedup | efficiency |
|---|---|---|---|---|
| Secuencial | 0 | 1869.45 | 1.0000 | — |
| Concurrente | 1 | 1820.02 | 1.0272 | 1.0272 |
| Concurrente | 2 | 951.15 | 1.9655 | 0.9827 |
| Concurrente | 4 | 510.57 | 3.6615 | 0.9154 |
| Concurrente | 8 | 397.33 | 4.7050 | 0.5881 |
| Concurrente | 12 | 387.66 | 4.8224 | 0.4019 |
| Concurrente | 16 | 382.95 | 4.8817 | 0.3051 |
| Concurrente | 24 | 384.63 | 4.8603 | 0.2025 |
| Concurrente | 32 | 384.12 | 4.8668 | 0.1521 |

Métricas del modelo entrenado (idénticas entre secuencial y
concurrente, verificación de equivalencia numérica): MSE
1,337,138,841.47; RMSE $36,566.91; MAE $12,216.60; R² 0.467884;
diferencia máxima de parámetros entre modelos: `0.000000000000`
(determinismo confirmado, ver `04-analisis-speedup-escalabilidad.md`).

### Dispersión (desviación estándar / CV)

El informe oficial no reporta `stddev_ms`/`cv` por configuración (solo
la media recortada de las 7 corridas). La rúbrica del punto (d) pide
explícitamente "tabla y estadística", por lo que estos dos campos
quedan como una mejora pendiente del informe, no como un dato
inventado aquí. Para completarlos hay dos caminos:

1. Recuperar los 7 tiempos individuales por configuración desde las
   capturas de consola/planillas del equipo (si se conservan) y
   calcular `stddev`/`cv` manualmente con las fórmulas de la sección
   anterior.
2. Volver a ejecutar el benchmark con la herramienta del repositorio
   (ver comando de reproducción abajo), que genera
   `results/benchmark/benchmark_runs.csv` (una fila por corrida
   individual) y `results/benchmark/speedup_summary.csv`/`.md` (con
   `stddev_ms` y `cv` ya calculados por configuración) — ver
   `07-observaciones-informe.md` para el texto a pegar en el
   informe.

### Reproducción exacta de la metodología del informe

El comando genérico de la sección "Cómo reproducir" usa los defaults
del repositorio (`-runs 10 -trim 0.1`). Para reproducir la metodología
exacta del informe (7 corridas, se descarta 1 mínimo + 1 máximo) hay
que ajustar ambos flags:

```bash
go run . -mode=benchmark -runs=7 -trim=0.15 -epochs 100 \
  -workers 1,2,4,8,12,16,24,32 -out results -dataset dataset/SPARCS_2022_clean_go.csv
```

`trim=0.15` es necesario porque `trimmedMean` recorta
`floor(n * fraction)` valores por extremo (`stats.go:27-43`): con
`n=7`, `floor(7 * 0.15) = 1` por lado (1 mínimo + 1 máximo, igual que
el informe). **Advertencia**: `-trim=0.1` con `-runs=7` recortaría
`floor(7 * 0.1) = floor(0.7) = 0` por extremo, es decir, usaría la
media simple de las 7 corridas sin descartar el mínimo/máximo —no
reproduce la metodología del informe.
