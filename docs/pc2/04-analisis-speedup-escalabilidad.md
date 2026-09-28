# 04 — Análisis de speedup, escalabilidad y trade-offs

Este documento fija el **marco de análisis** (fórmulas y qué comparar)
para interpretar los datos que produce `03-metodologia-benchmark.md`.
Los valores concretos deben completarse una vez corrido el benchmark
con el dataset real — los placeholders están marcados explícitamente.

## Ley de Amdahl

Si `f` es la fracción del trabajo total que es inherentemente
secuencial (no paralelizable) y `p` el número de workers:

```
S(p) = 1 / ((1 - f) + f/p)
```

`S(p)` es el speedup teórico máximo alcanzable con `p` workers. A
medida que `p → ∞`, `S(p) → 1/(1-f)`: existe un techo de speedup
determinado por la parte secuencial, sin importar cuántos workers se
agreguen.

En este algoritmo, la parte secuencial por época incluye: crear los
canales, encolar los jobs, cerrar el canal, el `wg.Wait()`, la
reducción determinista (`concurrent.go:150-163`, recorrido secuencial
de `partialResults` por `JobID`) y la actualización de pesos
(`concurrent.go:167-205`). La parte paralelizable es el cálculo de
gradientes parciales dentro de cada Worker.

## Estimar `f` con la métrica de Karp–Flatt

A diferencia de Amdahl (que requiere conocer `f` de antemano), la
métrica de Karp–Flatt permite **estimar** la fracción secuencial
experimental `e` a partir del speedup medido `S(p)` con `p` workers:

```
e = (1/S(p) - 1/p) / (1 - 1/p)
```

Si `e` se mantiene aproximadamente constante al variar `p`, el modelo
de Amdahl explica bien el comportamiento observado (la fracción
secuencial es intrínseca al algoritmo). Si `e` crece con `p`, hay
overhead adicional que Amdahl no captura (p. ej. contención de
memoria, overhead de scheduling de goroutines, saturación del bus de
memoria).

> TODO(equipo): calcular `e` para cada `p` medido (usar `Speedup` de
> `results/benchmark/speedup_summary.md`) y reportar si se mantiene
> estable o crece con `p`.

## Comportamiento esperado para este algoritmo

- **Memory-bandwidth bound**: el gradiente sobre ~1M+ filas es una
  operación con bajo cómputo por byte leído (mayormente sumas y
  multiplicaciones sobre floats leídos secuencialmente del slice
  `trainData`). Es esperable que el speedup se sature antes de
  alcanzar el número de CPUs lógicas, porque el ancho de banda de
  memoria (no el cómputo) se vuelve el cuello de botella al agregar
  más workers compitiendo por el mismo bus de memoria.
- **Overhead de spawn de goroutines y canales por época**: `trainConcurrent`
  crea `jobs`/`results` y lanza `workerCount` goroutines **en cada
  época** (`concurrent.go:104-123`), no una sola vez para todo el
  entrenamiento. Con `epochs` grande este overhead se repite
  `epochs` veces; para configuraciones con pocos jobs por worker o
  datasets pequeños, ese overhead puede dominar sobre el trabajo útil
  paralelizado.
- **Parte secuencial del reduce/update**: crece con el número de
  features, no con el número de filas, por lo que su peso relativo
  disminuye a medida que el dataset es más grande (mejor speedup
  esperado con datasets grandes que con datasets pequeños).
- **Hyperthreading más allá de núcleos físicos**: en CPUs con SMT/
  Hyperthreading, agregar workers más allá del número de núcleos
  físicos típicamente da retornos decrecientes marcados (dos hilos
  lógicos comparten unidades de ejecución del núcleo físico), y para
  cargas memory-bound puede incluso degradar el speedup por mayor
  contención de caché/memoria.

> TODO(equipo): contrastar lo anterior contra los datos medidos —
> ¿en qué `workers` se observa el pico de speedup? ¿coincide
> aproximadamente con `runtime.NumCPU()` físico o lógico de la
> máquina de prueba? ¿la eficiencia cae antes o después de ese punto?

## Trade-offs: secuencial vs. concurrente

| Aspecto | Secuencial (`sequential.go`) | Concurrente (`concurrent.go`) |
|---|---|---|
| Complejidad de código | Baja: un solo loop por época | Media: coordinación con canales, WaitGroup, particionamiento, reducción por JobID |
| Determinismo | Trivial (un solo orden de ejecución) | Garantizado por diseño (reducción por `JobID`, no por orden de llegada) — verificado en `TestConcurrentTrainingIsDeterministic` |
| Memoria | Un solo buffer de gradientes (`weightGradients`) | Un buffer de gradientes por Worker + buffers de canal (`jobs`, `results`) + slice `partialResults` |
| Overhead por época | Ninguno | Creación de canales + `workerCount` goroutines + sincronización (`wg.Wait()`) |
| Escalabilidad | No escala (single-threaded, mismo tiempo sin importar CPUs disponibles) | Escala hasta el punto de equilibrio (ver `05-recursos-punto-equilibrio.md`), limitada por ancho de banda de memoria y la parte secuencial del reduce/update |
| Cuándo conviene | Datasets pequeños, pocas épocas, o entornos de 1 sola CPU | Datasets grandes (~1M+ filas), múltiples CPUs disponibles, tiempo de entrenamiento es un cuello de botella real |

> TODO(equipo): completar con los valores medidos (speedup pico,
> `workers` de equilibrio, eficiencia en ese punto) citando
> `results/benchmark/speedup_summary.md` y `results/benchmark/equilibrium.md`.
