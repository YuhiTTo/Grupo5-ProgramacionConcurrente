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

Aplicando la fórmula a los speedups medidos (`03-metodologia-benchmark.md`):

| p | Speedup(p) | e (Karp–Flatt) |
|---|---|---|
| 2 | 1.9655 | 0.0176 |
| 4 | 3.6615 | 0.0308 |
| 8 | 4.7050 | 0.1000 |
| 12 | 4.8224 | 0.1353 |
| 16 | 4.8817 | 0.1518 |
| 24 | 4.8603 | 0.1712 |
| 32 | 4.8668 | 0.1799 |

`e` **no se mantiene constante**: crece de forma monótona con `p`
(0.0176 → 0.1799). Esto significa que el modelo de Amdahl puro (una
fracción secuencial fija `f`) **no explica** por sí solo la meseta de
speedup observada — si `f` fuera realmente ~0.02–0.03 (el valor que
sugiere `e` en `p=2`), Amdahl predeciría un speedup asintótico
`1/f ≈ 33–50`, muy por encima del ~4.88 observado como máximo. El
crecimiento de `e` indica que hay **overhead adicional dependiente de
`p`** que Amdahl no captura como fracción secuencial fija.

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

Contraste con los datos medidos (máquina de prueba: Ryzen 5 9600X, 6
núcleos físicos / 12 lógicos):

- El **pico de speedup** se observa en `workers=16` (4.8817x), pero la
  curva ya se aplana claramente a partir de `workers=8`
  (S(4)=3.6615 → S(8)=4.7050 → S(12)=4.8224 → S(16)=4.8817 →
  S(24)=4.8603 → S(32)=4.8668): entre 8 y 32 workers el speedup
  prácticamente no se mueve (4.70x a 4.88x, +3.8 % en un rango de 4x
  más workers), mientras que entre 4 y 8 workers ganó +28.5 % (3.66x
  a 4.70x).
- El **quiebre de la curva ocurre justo después de los 6 núcleos
  físicos**: `workers=4` todavía escala casi linealmente
  (E=0.9154), pero `workers=8` (ya por encima de los 6 núcleos
  físicos, usando hilos SMT) cae a E=0.5881, y de ahí en adelante la
  eficiencia sigue bajando monótonamente (0.4019 en 12, 0.3051 en 16,
  0.2025 en 24, 0.1521 en 32) aunque el speedup absoluto se mantenga
  casi plano. Esto es consistente con el comportamiento esperado de
  hyperthreading en carga memory-bound: los hilos lógicos adicionales
  (7 a 12) comparten unidades de ejecución de punto flotante/SIMD y
  ancho de banda de memoria con su núcleo físico, por lo que agregan
  paralelismo nominal sin agregar throughput real de cómputo.
- **Conclusión**: la meseta de speedup (~4.8–4.9x con 6 núcleos
  físicos disponibles) no se explica por una fracción secuencial fija
  (ver Karp–Flatt arriba); se explica por el límite físico de
  hardware (6 núcleos físicos, memoria/FP compartidos entre hilos SMT)
  combinado con el overhead de coordinación por época (creación de
  canales y goroutines, `wg.Wait()`) que crece con `p`.

## Trade-offs: secuencial vs. concurrente

| Aspecto | Secuencial (`sequential.go`) | Concurrente (`concurrent.go`) |
|---|---|---|
| Complejidad de código | Baja: un solo loop por época | Media: coordinación con canales, WaitGroup, particionamiento, reducción por JobID |
| Determinismo | Trivial (un solo orden de ejecución) | Garantizado por diseño (reducción por `JobID`, no por orden de llegada) — verificado en `TestConcurrentTrainingIsDeterministic` |
| Memoria | Un solo buffer de gradientes (`weightGradients`) | Un buffer de gradientes por Worker + buffers de canal (`jobs`, `results`) + slice `partialResults` |
| Overhead por época | Ninguno | Creación de canales + `workerCount` goroutines + sincronización (`wg.Wait()`) |
| Escalabilidad | No escala (single-threaded, mismo tiempo sin importar CPUs disponibles) | Escala hasta el punto de equilibrio (ver `05-recursos-punto-equilibrio.md`), limitada por ancho de banda de memoria y la parte secuencial del reduce/update |
| Cuándo conviene | Datasets pequeños, pocas épocas, o entornos de 1 sola CPU | Datasets grandes (~1M+ filas), múltiples CPUs disponibles, tiempo de entrenamiento es un cuello de botella real |

Con los valores medidos: el speedup pico es 4.8817x en `workers=16`,
pero el punto de equilibrio (menor `workers` que alcanza ≥95 % del
speedup máximo, ver `05-recursos-punto-equilibrio.md`) es
`workers=8` (4.7050x = 96.4 % de 4.8817x, E=0.5881). El informe
oficial eligió `workers=12` como punto de equilibrio (E=0.4019,
+2.5 % de speedup respecto a 8 workers a cambio de ~44 % más
`mallocs`); ambos criterios y su justificación se detallan en
`05-recursos-punto-equilibrio.md`. En cualquier caso, más allá de
~8-12 workers el costo en memoria/asignaciones sigue subiendo mientras
el speedup se mantiene esencialmente plano (4.70x–4.88x entre 8 y 32
workers) — la región de mayor retorno sobre la inversión de paralelismo
es `1 → 8` workers, no `8 → 32`.
