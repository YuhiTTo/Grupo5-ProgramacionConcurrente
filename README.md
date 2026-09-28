# Grupo5-ProgramacionConcurrente

Regresión Lineal (Gradient Descent) entrenada sobre el dataset SPARCS
de altas hospitalarias de NY, con dos implementaciones en Go —
**secuencial** y **concurrente** (Worker Pool + canales) — comparadas
en tiempo de entrenamiento, uso de recursos y escalabilidad. Proyecto
para la PC2 de Programación Concurrente (ver
`Documentos/CC65_PCs_TP-202620.pdf`).

Documentación completa de la evidencia de la PC2: [`docs/pc2/`](docs/pc2/README.md).

## Dataset

El CSV original (~947 MB) no se versiona por el límite de 100 MB de
GitHub (ver `.gitignore` y `dataset/README.md`). Descargar desde:

- Dataset original: <https://drive.google.com/file/d/1Tf4ubJF3OuRpqYnGQBM-nQEQsxjlvFGw/view?usp=sharing>
- Dataset limpio (formato usado por este código, `SPARCS_2022_clean_go.csv`): <https://drive.google.com/file/d/1cQwAvdyhqbVbZN5Wj8cJuv3X1kOPbJoh/view?usp=sharing>

Colocar el archivo limpio en `dataset/SPARCS_2022_clean_go.csv` (o
pasar otra ruta con `-dataset`). Alternativamente, se puede regenerar
el limpio desde el original con `-mode=clean` (ver `cleaning.go`).

## Cómo ejecutar

```bash
go run . [flags]
```

También es válido usar el binario legado `go run . cpu-profile`
(equivalente a `-mode=cpu-profile`).

### Flags (`config.go`)

| Flag | Default | Descripción |
|---|---|---|
| `-mode` | `all` | `quick`\|`benchmark`\|`resources`\|`cpu-profile`\|`clean`\|`all` |
| `-runs` | `10` | Ejecuciones por configuración en el benchmark formal |
| `-warmup` | `1` | Ejecuciones de calentamiento descartadas antes de medir |
| `-epochs` | `100` | Épocas de entrenamiento |
| `-workers` | `1,2,4,8,12,16,24,32` | Lista de workers a benchmarkear, separada por comas |
| `-trim` | `0.1` | Fracción recortada POR LADO en la media recortada (0.1 = 10% inferior + 10% superior) |
| `-out` | `results` | Directorio de salida para la evidencia persistida |
| `-dataset` | `dataset/SPARCS_2022_clean_go.csv` | Ruta al dataset limpio (CSV) |

### Modos

- `quick` — una comparación secuencial vs. concurrente (todas las
  CPUs lógicas) con verificación de equivalencia numérica.
- `benchmark` — benchmark formal (múltiples runs, warmup, media
  recortada, speedup, eficiencia, punto de equilibrio) para cada
  configuración de `-workers`.
- `resources` — perfil de uso de recursos (heap, allocs, GC,
  goroutines) por configuración.
- `cpu-profile` — perfil de CPU (`runtime/pprof`) de una corrida
  concurrente larga, analizable con `go tool pprof`.
- `clean` — regenera el dataset limpio a partir del CSV original
  (`cleaning.go`), sin correr el pipeline de regresión.
- `all` (default) — corre `quick` + `benchmark` + `resources` en
  secuencia.

Ejemplos:

```bash
go run . -mode=quick -epochs 100
go run . -mode=benchmark -runs 10 -epochs 100 -warmup 1 -workers 1,2,4,8,12,16,24,32
go run . -mode=resources -epochs 100
go run . -mode=cpu-profile -epochs 100
go run . -h   # lista todos los flags con sus defaults
```

## Tests

```bash
go test ./...              # suite completa
go test ./... -count=1 -v  # con detalle de cada test
go test -race ./...        # detección de condiciones de carrera (requiere cgo/gcc)
```

`go test -race` necesita un compilador C (gcc/clang vía cgo) en el
PATH; en Windows suele requerir MSYS2/MinGW o similar.

## Resultados (`results/`)

Generados por `-mode=all`/`benchmark`/`resources`/`cpu-profile` bajo
`-out` (default `results/`):

```
results/
├── environment.md                    # metadatos de entorno (Go, SO, CPUs, config de la corrida)
├── benchmark/
│   ├── benchmark_runs.csv            # cada corrida individual
│   ├── speedup_summary.{csv,md}      # estadísticas + speedup + eficiencia por configuración
│   └── equilibrium.md                # punto de equilibrio y punto de caída de eficiencia
├── resources/
│   └── resources.csv                 # heap, allocs, GC, goroutines por configuración
└── cpu/
    └── cpu.prof                      # perfil pprof (no versionado, ver .gitignore)
```

## Verificación formal (Promela / Spin)

El modelo de sincronización del pipeline concurrente
(`promela/regression_workers.pml`) se verifica con Spin: ausencia de
deadlock, invalid end states, violaciones de aserción, y 3
propiedades LTL (`safe_update`, `termination`, `mutex`). Ver
[`docs/pc2/02-modelo-promela.md`](docs/pc2/02-modelo-promela.md) para
el mapeo Promela↔Go y cómo ejecutar `promela/run_spin.sh` /
`run_spin.ps1`.

## Documentación PC2

Ver [`docs/pc2/README.md`](docs/pc2/README.md) para el índice completo
y el mapeo de cada punto de la rúbrica a su evidencia.
