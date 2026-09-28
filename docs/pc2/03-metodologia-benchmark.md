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

> TODO(equipo): correr el comando de arriba con el dataset real
> (`dataset/SPARCS_2022_clean_go.csv`, ver README raíz para la
> descarga) y pegar aquí el contenido de
> `results/benchmark/speedup_summary.md`.

Columnas esperadas (igual a `speedupSummaryHeader`, `report.go:123-126`):

| mode | workers | runs | mean_ms | trimmed_mean_ms | median_ms | stddev_ms | min_ms | max_ms | cv | speedup | efficiency |
|---|---|---|---|---|---|---|---|---|---|---|---|
| Secuencial | 0 | ... | ... | ... | ... | ... | ... | ... | ... | 1.0000 | 1.0000 |
| Concurrente | 1 | ... | ... | ... | ... | ... | ... | ... | ... | ... | ... |
| Concurrente | 2 | ... | ... | ... | ... | ... | ... | ... | ... | ... | ... |
| ... | ... | ... | ... | ... | ... | ... | ... | ... | ... | ... | ... |
