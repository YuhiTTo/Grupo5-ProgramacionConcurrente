# Grupo5-ProgramacionConcurrente

Regresión lineal multivariable entrenada con **Batch Gradient Descent** sobre
más de 2 millones de altas hospitalarias (SPARCS 2022, Nueva York), en dos
versiones escritas en Go: una **secuencial** y otra **concurrente** (Worker
Pool con goroutines, canales y `sync.WaitGroup`). El proyecto compara ambas
en tiempo, speedup, escalabilidad y uso de recursos, y verifica formalmente la
sincronización con **Promela/Spin**.

Curso: CC65 Programación Concurrente (UPC, 2026-20). Enunciado:
[`Documentos/CC65_PCs_TP-202620.pdf`](Documentos/CC65_PCs_TP-202620.pdf).

| Integrante | Código | Cuenta de GitHub |
|---|---|---|
| Lucero Salome Manchay Paredes | U202216120 | `YuhiTTo` |
| José Antonio Mayhua Hinostroza | U202218044 | `SeuNg720p` |
| Jhamil Brijan Peña Cardenas | U201714492 | `Jaed69` |

---

## Qué hacemos

### El problema

A partir de variables clínicas y administrativas de cada hospitalización
(días de estancia, severidad, grupo de edad, tipo de admisión, categoría
diagnóstica, tipo de pago, ingreso por emergencia) el modelo estima el
**costo total de la atención** (`Total Costs`). El caso de uso se enmarca en
el ODS 3 (Salud y bienestar): apoyar la planificación presupuestaria de los
hospitales.

### El pipeline

1. **Limpieza** (`cleaning.go`): 2,103,433 registros originales, de los cuales
   se conservan 2,103,432.
2. **Preprocesamiento** (`preprocessing.go`, `scaling.go`): codificación
   one-hot, lo que da 49 características; partición 80/20 reproducible
   (1,682,537 registros de entrenamiento); estandarización calculada solo con
   el conjunto de entrenamiento.
3. **Entrenamiento**: 100 épocas de Batch Gradient Descent con learning rate
   0.01, en versión secuencial (`sequential.go`) y concurrente
   (`concurrent.go`).
4. **Evaluación**: MSE, RMSE, MAE y R² sobre el conjunto de prueba, más una
   verificación de que ambas versiones producen el mismo modelo
   (`equivalence.go`).

### El diseño concurrente

```
                 ┌──────────── jobs (chan) ────────────┐
Coordinador ──►  │ job 0 │ job 1 │ ... │ job N-1        │ ──► Worker 1..W
(trainConcurrent)└─────────────────────────────────────┘     (gradiente parcial,
       ▲                                                        solo lectura de pesos)
       │                 results (chan)                              │
       └──────────── reducción por JobID ◄───────────────────────────┘
                     wg.Wait() → actualización de pesos (un solo flujo)
```

- En cada época el conjunto de entrenamiento se divide en `workers × 4` jobs.
- Cada worker calcula un **gradiente parcial** sobre su porción y solo
  **lee** los pesos.
- El coordinador espera con `sync.WaitGroup`, reduce los resultados en
  orden de `JobID` (por eso el resultado es determinista) y recién ahí
  actualiza los pesos.
- **No hace falta `sync.Mutex`**: ningún worker escribe estado compartido.
  Spin lo demuestra con la propiedad LTL `mutex`.

### Resultados principales

Medidos en un AMD Ryzen 5 9600X (6 núcleos físicos, 12 lógicos), con 7
ejecuciones por configuración y media recortada:

| Configuración | Tiempo | Speedup | Eficiencia |
|---|---:|---:|---:|
| Secuencial | 1869.45 ms | 1.00x | – |
| 4 workers | 510.57 ms | 3.66x | 0.92 |
| 8 workers | 397.33 ms | 4.71x | 0.59 |
| 12 workers | 387.66 ms | 4.82x | 0.40 |
| 16 workers | 382.95 ms | 4.88x | 0.31 |

- **Mismo modelo**: la diferencia máxima entre parámetros es 0 y ambas
  versiones obtienen R² = 0.4679.
- **Escalabilidad**: el speedup se estanca cerca de 4.9x. La métrica de
  Karp–Flatt muestra que el límite viene del hardware (6 núcleos físicos,
  ancho de banda de memoria), no de código secuencial.
- **Punto de equilibrio**: 8 workers según el criterio de eficiencia (96 % del
  speedup máximo) y 12 workers como elección práctica.
- **Spin**: todas las variantes (2/4, 3/4 y 4/4) terminan con `errors: 0` en
  seguridad y en las propiedades `safe_update`, `termination` y `mutex`.

El análisis completo está en [`docs/pc2/`](docs/pc2/README.md).

---

## Cómo ejecutarlo

### Requisitos

- **Go 1.27 o superior.** No hay dependencias externas: solo se usa la
  librería estándar.
- Unos 250 MB libres para el dataset limpio (~950 MB más si también querés
  el original).
- Opcional: **Docker**, para la verificación con Spin.

### Inicio rápido

```bash
git clone https://github.com/YuhiTTo/Grupo5-ProgramacionConcurrente.git
cd Grupo5-ProgramacionConcurrente
go run . -mode=quick
```

La primera vez, el programa descarga el dataset limpio (~244 MB) de forma
automática y verificada, entrena ambas versiones e imprime tiempos, speedup,
métricas y el chequeo de equivalencia.

### Modos

| Comando | Qué hace | Duración aprox. |
|---|---|---|
| `go run . -mode=quick` | Una comparación secuencial vs. concurrente (todas las CPUs) | < 1 min |
| `go run . -mode=benchmark` | Benchmark formal: varias ejecuciones por cada cantidad de workers, media recortada, speedup, eficiencia y punto de equilibrio | varios minutos |
| `go run . -mode=resources` | Memoria, asignaciones, GC y goroutines por configuración | ~1–2 min |
| `go run . -mode=cpu-profile` | Perfil de CPU para `go tool pprof` | ~1 min |
| `go run . -mode=all` | `quick` + `benchmark` + `resources` (modo por defecto) | varios minutos |
| `go run . -mode=download` | Solo descarga y verifica el dataset (`-download=clean\|raw\|all`) | según la red |
| `go run . -mode=clean` | Regenera el dataset limpio desde el original | ~1 min |

Para mediciones confiables, corré el benchmark con la PC enchufada y sin otras
aplicaciones abiertas.

**Reproducir la metodología del informe** (7 ejecuciones, descartando la
mínima y la máxima):

```bash
go run . -mode=benchmark -runs=7 -trim=0.15
```

### Flags

| Flag | Default | Descripción |
|---|---|---|
| `-mode` | `all` | `quick`, `benchmark`, `resources`, `cpu-profile`, `clean`, `all` o `download` |
| `-runs` | `10` | Ejecuciones medidas por configuración |
| `-warmup` | `1` | Ejecuciones de calentamiento descartadas |
| `-epochs` | `100` | Épocas de entrenamiento |
| `-workers` | `1,2,4,8,12,16,24,32` | Cantidades de workers a evaluar |
| `-trim` | `0.1` | Fracción recortada **por lado** en la media recortada (con 7 ejecuciones usar `0.15`; con `0.1` no se recorta nada) |
| `-out` | `results` | Carpeta de salida |
| `-dataset` | `dataset/SPARCS_2022_clean_go.csv` | Ruta al dataset limpio |
| `-download` | `clean` | Con `-mode=download`: `clean`, `raw` o `all` |
| `-no-download` | `false` | Desactiva la descarga automática |

`go run . -h` lista todos los flags. `go run . cpu-profile` sigue funcionando
como alias de `-mode=cpu-profile`.

### Resultados generados

```
results/
├── environment.md              # Go, SO, CPUs y configuración de la corrida
├── benchmark/
│   ├── benchmark_runs.csv      # cada ejecución individual
│   ├── speedup_summary.csv/.md # media recortada, desviación, CV, speedup, eficiencia
│   └── equilibrium.md          # punto de equilibrio y caída de eficiencia
├── resources/resources.csv     # heap, asignaciones, GC, goroutines
├── cpu/cpu.prof                # perfil pprof (no versionado)
└── promela/                    # salidas de Spin
```

Para analizar el perfil de CPU: `go tool pprof -top results/cpu/cpu.prof`.

### Dataset

El CSV no se sube al repositorio por su tamaño. La descarga automática usa
solo HTTPS contra un host permitido y verifica el tamaño exacto y el SHA-256
antes de aceptar el archivo. Para descargarlo a mano y verificarlo, o si el
archivo de origen cambia, ver [`dataset/README.md`](dataset/README.md).

### Tests

```bash
go test ./...          # suite completa
go test -race ./...    # detector de condiciones de carrera (requiere gcc/cgo)
```

### Verificación formal con Spin

El modelo [`promela/regression_workers.pml`](promela/regression_workers.pml)
abstrae la sincronización del Worker Pool. Sin Spin instalado, se puede correr
con Docker:

```bash
docker run --rm -v "$PWD":/work -w /work debian:stable-slim bash -ec \
  'apt-get update -qq && apt-get install -y -qq spin gcc libc6-dev && \
   for v in "3 4" "2 4" "4 4"; do bash promela/run_spin.sh $v; done'
```

Con Spin y gcc instalados localmente: `./promela/run_spin.sh 3 4` o
`./promela/run_spin.ps1 -Workers 3 -Jobs 4`. Los scripts fallan si `pan`
reporta errores.

---

## Estructura del repositorio

| Ruta | Contenido |
|---|---|
| `main.go`, `config.go` | Punto de entrada, modos y flags |
| `dataset.go`, `cleaning.go`, `dataset_download.go` | Lectura, limpieza y descarga segura del dataset |
| `preprocessing.go`, `scaling.go` | Codificación, partición y estandarización |
| `regression.go`, `sequential.go`, `concurrent.go` | Modelo y entrenamiento secuencial/concurrente |
| `benchmark.go`, `stats.go`, `report.go` | Benchmark, estadística y reportes |
| `resource_profile.go`, `cpu_profile.go` | Perfiles de memoria y CPU |
| `equivalence.go` | Verificación de que ambas versiones dan el mismo modelo |
| `*_test.go` | Tests |
| `promela/` | Modelo Promela y scripts de Spin |
| `results/` | Evidencia generada |
| `docs/pc2/` | Documentación por punto de la rúbrica |
| `Documentos/` | Enunciado e informes entregados |

## Documentación

- [`docs/pc2/README.md`](docs/pc2/README.md): índice y mapeo de cada punto de
  la rúbrica a su evidencia.
- [`docs/pc2/01-algoritmo-concurrente.md`](docs/pc2/01-algoritmo-concurrente.md):
  algoritmo y sincronización.
- [`docs/pc2/02-modelo-promela.md`](docs/pc2/02-modelo-promela.md): modelo
  Promela, propiedades LTL y resultados de Spin.
- [`docs/pc2/04-analisis-speedup-escalabilidad.md`](docs/pc2/04-analisis-speedup-escalabilidad.md)
  y [`05-recursos-punto-equilibrio.md`](docs/pc2/05-recursos-punto-equilibrio.md):
  análisis de rendimiento.

## Flujo de trabajo

Git Flow: cada cambio se desarrolla en una rama `feature/*`, se integra en
`develop` mediante un pull request y de ahí pasa a `main`.
