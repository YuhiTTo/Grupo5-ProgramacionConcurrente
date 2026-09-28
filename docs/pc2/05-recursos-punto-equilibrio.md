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

> TODO(equipo): correr `go run . -mode=all` (o `-mode=resources`) con
> el dataset real y pegar aquí:
> - El contenido de `results/resources/resources.csv` (o una tabla
>   resumida).
> - El contenido de `results/benchmark/equilibrium.md`.
> - La salida de `go tool pprof -top results/cpu/cpu.prof` (top 10
>   funciones).
