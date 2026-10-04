# 01 — Verificación formal en Spin (TP, punto q)

El enunciado del TP pide una **verificación formal en Spin de la ausencia de
deadlocks y de la exclusión mutua**. Este documento resume qué se verifica,
cómo se relaciona con el código Go y cuáles fueron los resultados. El modelo
y su mapeo detallado a Go están en
[`docs/pc2/02-modelo-promela.md`](../pc2/02-modelo-promela.md).

## Qué se verifica

| Propiedad | Cómo se verifica en Spin | Qué garantiza en Go |
|---|---|---|
| **Ausencia de deadlock** | Corrida de seguridad (`gcc -DSAFETY -DNOCLAIM`, `./pan`): Spin explora todos los entrelazados y reporta cualquier *invalid end state*, es decir, un estado final en el que algún proceso quedó bloqueado | Ningún `gradientWorker` queda esperando para siempre en `jobs`, y `trainConcurrent` no se bloquea en `wg.Wait()` |
| **Exclusión mutua** | LTL `mutex`: `[] (in_update <= 1)` | A lo sumo un flujo escribe `model.Weights`/`model.Bias` a la vez; por eso no hace falta `sync.Mutex` |
| Actualización segura | LTL `safe_update`: `[] (updating -> result_count == NUM_JOBS)` | Los pesos solo se actualizan después de reunir todos los gradientes parciales (`wg.Wait()`) |
| Terminación | LTL `termination`: `<> done` (con búsqueda de ciclos de aceptación, `-a`) | La época siempre termina; no hay livelock |
| Integridad del trabajo | Aserciones: cada job se procesa una sola vez y cada resultado se recibe una sola vez | El particionamiento y la reducción por `JobID` no pierden ni duplican trabajo |

## Resultados del modelo correcto

`promela/regression_workers.pml`, Spin 6.5.2. Salidas completas en
`results/promela/w<N>_j<M>_*.txt`.

| Workers / jobs | Estados almacenados | Seguridad (deadlock) | `mutex` | `safe_update` | `termination` |
|---|---:|:---:|:---:|:---:|:---:|
| 2 / 4 | 12,358 | 0 errores | 0 errores | 0 errores | 0 errores |
| 3 / 4 | 145,563 | 0 errores | 0 errores | 0 errores | 0 errores |
| 4 / 4 | 1,080,202 | 0 errores | 0 errores | 0 errores | 0 errores |

La variante 4/8 superó los 34 millones de estados y 3.2 GB de memoria sin
terminar (explosión del espacio de estados). Por eso la verificación
exhaustiva se hace con N pequeño, que alcanza para cubrir todos los
entrelazados de la lógica de sincronización.

## Los mutantes: la verificación detecta errores reales

Un resultado de `0 errores` solo tiene valor si Spin **sí** reporta errores
cuando la sincronización está mal. Para demostrarlo se crearon dos
**mutantes**: copias del modelo con un único defecto introducido a propósito,
marcado con `MUTANT:` en el código.

| Mutante | Defecto introducido | Equivalente en Go | Chequeo | Resultado |
|---|---|---|---|---|
| `regression_workers_race.pml` | Cada Worker, después de enviar su resultado, también entra a la sección de actualización de pesos | `gradientWorker` escribiendo `model.Weights` sin sincronización | LTL `mutex` | **Violada** (`errors: 1`): dos procesos dentro de la sección crítica a la vez (`in_update` llega a 2), contraejemplo a profundidad 213 |
| `regression_workers_race.pml` | (el mismo) | Actualizar pesos sin esperar `wg.Wait()` | LTL `safe_update` | **Violada** (`errors: 1`): un Worker marca `updating` antes de que el Coordinator reciba los `NUM_JOBS` resultados, contraejemplo a profundidad 33 |
| `regression_workers_deadlock.pml` | El Coordinator envía un `STOP` menos que la cantidad de Workers | Un Worker que nunca recibe la señal de fin, análogo a no cerrar `jobs` (en Go se bloquearían todos los workers en `for job := range jobs`; el mutante modela el caso mínimo de uno) | Seguridad | **Deadlock detectado** (`errors: 1`, *invalid end state* a profundidad 106) |

Cada corrida guarda la reproducción del contraejemplo (`spin -t -p`) en
`results/promela/mutants/<mutante>_<chequeo>_trail.txt`, que muestra paso a
paso el entrelazado que rompe la propiedad.

Conclusión: el mismo procedimiento que **rechaza** los mutantes **acepta** el
modelo correcto, así que el resultado `0 errores` del modelo correcto es
evidencia real de que el diseño no tiene deadlocks y respeta la exclusión
mutua.

## Evidencia complementaria en Go

Spin verifica la lógica de sincronización abstracta. Para el código real se
ejecutó el detector de carreras de Go:

```bash
docker run --rm -v "$PWD":/src:ro -w /src golang:1.27 go test -race -count=1 ./...
```

Resultado (Go 1.27.1 linux/amd64): `ok`, sin carreras detectadas. La salida
completa está en [`results/race/go_test_race.txt`](../../results/race/go_test_race.txt)
(80 pruebas en PASS). `equivalence_test.go` ejecuta `trainConcurrent` con 1, 2,
4 y 8 workers, de modo que el detector cubrió el Worker Pool.

## Cómo reproducir

Requiere Docker (o Spin y gcc instalados localmente).

```bash
# Modelo correcto: seguridad + 3 propiedades LTL por variante
docker run --rm -v "$PWD":/work -w /work debian:stable-slim bash -ec \
  'apt-get update -qq && apt-get install -y -qq spin gcc libc6-dev && \
   for v in "2 4" "3 4" "4 4"; do bash promela/run_spin.sh $v; done'

# Mutantes: el script termina con código 0 solo si Spin detecta todos
docker run --rm -v "$PWD":/work -w /work debian:stable-slim bash -ec \
  'apt-get update -qq && apt-get install -y -qq spin gcc libc6-dev && \
   bash promela/run_spin_mutants.sh'
```

`run_spin.sh` falla si alguna corrida del modelo correcto reporta errores, y
`run_spin_mutants.sh` falla si algún mutante **no** es detectado.
