# 05 — Uso de recursos y punto de equilibrio

## Métricas capturadas

`measureResourceUsage` (`resource_profile.go:30-110`) samplea, cada
10ms mientras corre el entrenamiento (además de antes/después),
`runtime.MemStats` y `runtime.NumGoroutine()`, y devuelve por corrida:

- `ElapsedMS` — duración de la corrida en milisegundos.
- `PeakHeapMB` — pico de `HeapAlloc` observado (heap vivo).
- `AllocatedMB` — `TotalAlloc` (después) - `TotalAlloc` (antes): total
  asignado durante la corrida (incluye memoria ya liberada por el GC,
  a diferencia de `PeakHeapMB`).
- `Mallocs` — número de asignaciones (`runtime.MemStats.Mallocs`).
- `GCCount` — número de ciclos de GC ejecutados durante la corrida.
- `PeakGoroutines` — pico de goroutines vivas (coordinator + workers).

`profileSequentialResources`/`profileConcurrentResources`
(`resource_profile.go:112-251`) promedian estas métricas sobre
`resourceRuns` corridas (3, fijo en `main.go:643`) por configuración,
y `writeResourcesCSV` (`report.go:258-308`) las persiste en
`results/resources/resources.csv` con columnas:

```
workers, elapsed_ms, peak_heap_mb, total_alloc_mb, mallocs, num_gc, peak_goroutines
```

## Perfil de CPU (`cpu.prof`)

`-mode=cpu-profile` (`cpu_profile.go`) entrena un modelo concurrente
con todas las CPUs lógicas (`runtime.NumCPU()`) durante 3000 épocas
bajo `runtime/pprof`, y guarda el perfil en
`results/cpu/cpu.prof` (no versionado — ver `.gitignore`, binario
pesado y no reproducible bit a bit entre corridas). Para analizarlo:

```bash
go tool pprof -top results/cpu/cpu.prof
```

Esto lista las funciones que más tiempo de CPU consumieron (útil para
confirmar dónde se concentra el costo: `accumulateSampleGradient` en
`regression.go` debería dominar si el algoritmo está bien
paralelizado, y no el overhead de canales/goroutines).

## Definición del punto de equilibrio

Implementada en `findEquilibrium` (`report.go:314-340`): el **menor**
número de workers (orden ascendente) cuyo speedup alcanza el 95% del
speedup máximo observado entre todas las configuraciones medidas
(`threshold = 0.95`, llamado desde `main.go:599-600`). Es decir,
agregar más workers después de ese punto da como máximo un 5% más de
speedup — el "punto de equilibrio" entre beneficio marginal y costo
de más paralelismo (más goroutines, más contención, más memoria).

Si el speedup máximo observado es `<= 0` (todas las configuraciones
igual o más lentas que el secuencial), `findEquilibrium` retorna
`found = false`: no hay un punto de equilibrio significativo que
reportar (evita reportar falsamente "workers=1" como equilibrio
cuando en realidad ningún nivel de paralelismo ayudó).

Complementariamente, `findEfficiencyDrop` (`report.go:345-359`)
reporta el menor número de workers cuya `Efficiency` cae por debajo
de `0.5` (`threshold = 0.5`, `main.go:602-603`) — es decir, a partir
de qué punto se obtiene menos de 50% de la ganancia teórica lineal
por worker agregado.

Ambos se persisten en `results/benchmark/equilibrium.md`
(`writeEquilibriumMarkdown`, `report.go:363-410`).

## Cómo leer los resultados

1. Abrir `results/resources/resources.csv` — comparar `peak_heap_mb`
   y `mallocs` entre secuencial (`workers=0`) y cada configuración
   concurrente: más workers implica más buffers de gradientes locales
   y más entradas en los canales, por lo que se espera que
   `peak_heap_mb`/`mallocs` crezcan (moderadamente) con `workers`.
2. Abrir `results/benchmark/equilibrium.md` — el número de workers
   reportado es el punto a partir del cual seguir agregando workers
   no se justifica por el beneficio en tiempo de entrenamiento
   obtenido, considerando el costo adicional en recursos.
3. Cruzar ambos: si el punto de equilibrio (`0.95x` del speedup
   máximo) llega ANTES que un salto notable en `peak_heap_mb` o
   `num_gc`, el trade-off recursos-vs-tiempo es claramente favorable
   hasta ese punto. Si el salto en recursos ocurre antes, puede
   convenir quedarse con menos workers de los que da el punto de
   equilibrio puro de speedup.

## Resultados

Valores oficiales del informe (`resource_profile`, 3 corridas por
configuración, mismo entorno que `03-metodologia-benchmark.md`):

| workers | PeakHeap MB | Alloc MB | Mallocs | GC |
|---|---|---|---|---|
| Secuencial (0) | 141.92 | 0.04 | 108 | 0 |
| 1 | 142.15 | 0.27 | 1,109 | 0 |
| 2 | 142.36 | 0.49 | 1,609 | 0 |
| 4 | 142.81 | 0.93 | 2,609 | 0 |
| 8 | 143.67 | 1.80 | 4,610 | 0 |
| 12 | 144.54 | 2.67 | 6,625 | 0 |
| 16 | 145.35 | 3.47 | 8,633 | 0 |
| 24 | 147.07 | 5.19 | 12,622 | 0 |
| 32 | 148.76 | 6.88 | 16,665 | 0 |

`GCCount = 0` en todas las configuraciones: con 100 épocas el heap
asignado por corrida no llega a disparar un ciclo de GC (el pico de
heap crece de forma moderada y previsible con `workers`, de 141.92 MB
a 148.76 MB entre secuencial y 32 workers, +4.8 %), consistente con lo
esperado en la sección anterior (más buffers de gradientes locales y
más entradas de canal por worker).

### Punto de equilibrio (criterio del repositorio, `findEquilibrium`)

- **Speedup máximo observado**: 4.8817x (`workers=16`).
- **Umbral 95 %**: `4.8817 × 0.95 = 4.6376`.
- **Menor `workers` que alcanza el umbral**: `workers=8` (4.7050x =
  96.4 % del máximo) — es el primer valor, en orden ascendente, que
  supera 4.6376.
- **Caída de eficiencia (`findEfficiencyDrop`, umbral 0.5)**: la
  eficiencia cae por debajo de 0.5 por primera vez en `workers=12`
  (E=0.4019; `workers=8` todavía tiene E=0.5881 ≥ 0.5). Es decir,
  `workers=8` es también el último punto con `Efficiency ≥ 0.5`.

Bajo las reglas automáticas del repositorio, ambos criterios
(`findEquilibrium` al 95 % y `findEfficiencyDrop` al 50 %) coinciden
en señalar **`workers=8`** como el punto de equilibrio: es el menor
`workers` que alcanza el 95 % del speedup máximo, y a la vez el mayor
`workers` que todavía mantiene una eficiencia ≥ 0.5.

### Comparación con la elección del informe (`workers=12`)

El informe oficial eligió `workers=12` (coincide con el número de
núcleos lógicos de la máquina de prueba) como punto de equilibrio,
justificándolo así: pasar de 12 a 16 workers da solo +1.21 % de
ganancia de tiempo (`(4.8817-4.8224)/4.8224 ≈ 0.0123`, informe
redondea a +1.21 %) a cambio de +30 % en asignaciones/`mallocs`
(`(8,633-6,625)/6,625 ≈ 0.303`), con la eficiencia cayendo de 0.4019 a
0.3051. Ese argumento es válido como comparación **12 vs. 16**, pero
no contempla el tramo **8 vs. 12**, donde el mismo patrón de
diminishing returns ya es visible:

| Cambio | Δ speedup | Δ mallocs | Δ eficiencia |
|---|---|---|---|
| 8 → 12 | +2.50 % (4.7050→4.8224) | +43.7 % (4,610→6,625) | 0.5881 → 0.4019 |
| 12 → 16 | +1.21 % (4.8224→4.8817) | +30.3 % (6,625→8,633) | 0.4019 → 0.3051 |

**Recomendación para el informe** (texto listo para pegar en
`07-observaciones-informe.md`): presentar ambos criterios en vez de
uno solo — `workers=8` como el **equilibrio eficiencia-óptima** (según
las reglas automáticas del propio repositorio: 96.4 % del speedup pico
con 31 % menos mallocs que 12 workers y E=0.588 ≥ 0.5), y
`workers=12` como una **elección orientada a throughput** justificada
explícitamente porque coincide con el número de CPUs lógicas de la
máquina y compra un +2.5 % adicional de velocidad a cambio de más
memoria/asignaciones. Esto no invalida la elección de 12 workers del
informe; la complementa con el criterio cuantitativo que el propio
código del repositorio ya implementa (`findEquilibrium`/
`findEfficiencyDrop`, `report.go:314-359`).

### Perfil de CPU (`cpu.prof`)

El informe incluye evidencia visual de uso de CPU (Figura 11) para una
corrida con `workers=12` y `epochs=3000`: ~92 % de uso de CPU en el
Administrador de tareas de Windows, distribuido entre los 12 CPUs
lógicos. No se dispone del archivo `cpu.prof` binario en el
repositorio (no versionado, ver `.gitignore`) ni de la salida textual
de `go tool pprof -top`; quien quiera reproducir el top de funciones
por tiempo de CPU debe correr:

```bash
go run . -mode=cpu-profile -epochs 100
go tool pprof -top results/cpu/cpu.prof
```

y pegar la salida real — no se inventa aquí una lista de funciones que
no fue medida en este entorno.
