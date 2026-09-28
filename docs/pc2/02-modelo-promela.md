# 02 — Modelo Promela

## Qué se modela

`promela/regression_workers.pml` modela la **sincronización** del
pipeline concurrente descrito en `01-algoritmo-concurrente.md`, no el
cómputo matemático (no hay ~2M de filas ni gradient descent real en
el modelo). El objetivo es verificar, con model checking exhaustivo
(Spin/pan explora todos los entrelazados posibles de las goroutines),
que:

1. Cada job se procesa exactamente una vez.
2. Cada resultado se recibe exactamente una vez.
3. El Coordinator nunca actualiza los pesos antes de recibir **todos**
   los resultados.
4. Solo el Coordinator entra a la sección crítica de actualización de
   pesos (exclusión mutua, sin necesidad de un Mutex explícito).
5. El sistema siempre termina (no hay deadlock ni livelock).

## Mapeo Promela ↔ Go

| Promela | Go | Descripción |
|---|---|---|
| `proctype Worker` | `gradientWorker` (`concurrent.go:21`) | Una goroutine Worker por cada `run Worker(i)` |
| `chan jobs` | `chan GradientJob` (`concurrent.go:104`) | Canal buffered de trabajo |
| `chan results` | `chan GradientResult` (`concurrent.go:107`) | Canal buffered de resultados parciales |
| `jobs?job_id` / `STOP` | `for job := range jobs` + `close(jobs)` (`concurrent.go:30`, `:148`) | Consumo hasta cierre del canal (`STOP` modela la señal de fin) |
| `processed[job_id]` | (implícito: cada job se asigna a un único chunk) | Verifica que el particionamiento no reprocese un job |
| `init` (proceso) | Cuerpo de `trainConcurrent` fuera de las goroutines (`concurrent.go:60-224`) | El "Coordinator" |
| Reducer (`results?result_id`, `received[]`) | `partialResults[result.JobID] = result` (`concurrent.go:157-163`) | Recepción y consolidación por `JobID` |
| `assert(result_count == NUM_JOBS)` antes de actualizar pesos | `wg.Wait()` antes de la actualización (`concurrent.go:165`) | Invariante: no actualizar con datos incompletos |
| `in_update` / `updating` | Región de escritura de `model.Weights`/`model.Bias` (`concurrent.go:167-205`) | Sección crítica modelada explícitamente para verificar mutex |
| `done` | Fin de una época (`concurrent.go` fin del `for epoch` en una iteración) | Usado solo para la propiedad de terminación |

`NUM_WORKERS` y `NUM_JOBS` son análogos a `workerCount` y
`actualJobCount` en Go, pero fijos por corrida de verificación (Spin
no puede explorar con datos reales de 2M de filas; en cambio,
verifica exhaustivamente TODOS los entrelazados posibles para un N
pequeño, lo cual es más fuerte que un test con datos reales para
demostrar ausencia de condiciones de carrera en la lógica de
sincronización).

## Aserciones (seguridad)

Ya presentes en el modelo original (sin cambios de comportamiento):

- `assert(job_id < NUM_JOBS)` — todo job recibido es válido.
- `assert(processed[job_id] == 0)` seguido de `processed[job_id] = 1`
  (dentro de `atomic`) — ningún job se procesa dos veces.
- `assert(received[result_id] == 0)` — ningún resultado se recibe dos
  veces.
- `assert(result_count == NUM_JOBS)` — TODOS los resultados llegaron
  antes de continuar.
- `assert(processed[i] == 1)` / `assert(received[i] == 1)` para cada
  `i` — cobertura completa de jobs.
- `assert(weights_updated)` — la actualización efectivamente ocurrió.

Estas se verifican con `pan -a` (deadlock + invalid end states +
assertion violations), ver `results/promela/verificacion_spin.txt`
(corrida existente sobre una versión anterior del modelo, sin las
propiedades LTL nuevas): **0 errores**, 143799 estados explorados,
profundidad 120.

```
State-vector 80 byte, depth reached 120, errors: 0
   143799 states, stored
    97465 states, matched
   241264 transitions (= stored+matched)
```

## Propiedades LTL (nuevas)

Agregadas en `promela/regression_workers.pml`:

```promela
ltl safe_update  { [] (updating -> result_count == NUM_JOBS) }
ltl termination  { <> done }
ltl mutex        { [] (in_update <= 1) }
```

- **`safe_update`**: en todo momento (`[]`, "always"), si el
  Coordinator está actualizando pesos (`updating`), entonces ya se
  recibieron todos los resultados. Formaliza el invariante central de
  `wg.Wait()` antes de tocar `model.Weights`/`model.Bias`.
- **`termination`**: eventualmente (`<>`, "eventually") el sistema
  llega a `done`. Ausencia de deadlock/livelock para este modelo (una
  época).
- **`mutex`**: en todo momento, a lo sumo un proceso está dentro de la
  sección crítica de actualización (`in_update <= 1`). Verifica
  formalmente que un `sync.Mutex` explícito en Go sería redundante:
  la exclusión mutua ya está garantizada por el diseño (ver
  `01-algoritmo-concurrente.md`, sección "Por qué no hace falta un
  Mutex").

## Cómo ejecutar

Requiere `spin` y `gcc` instalados (no disponibles en el entorno donde
se escribió este documento — ver nota abajo).

```bash
# Linux/macOS/WSL/Git Bash
./promela/run_spin.sh              # NUM_WORKERS=3 NUM_JOBS=4 (defaults del modelo)
./promela/run_spin.sh 2 4          # NUM_WORKERS=2 NUM_JOBS=4
./promela/run_spin.sh 4 4          # NUM_WORKERS=4 NUM_JOBS=4
```

```powershell
# Windows PowerShell (con spin/gcc en el PATH, p. ej. vía MSYS2)
./promela/run_spin.ps1
./promela/run_spin.ps1 -Workers 2 -Jobs 4
./promela/run_spin.ps1 -Workers 4 -Jobs 4
```

Cada corrida hace, por variante:
1. `spin -a -DNUM_WORKERS=<n> -DNUM_JOBS=<m> regression_workers.pml`
2. `gcc -DSAFETY -DNOCLAIM -o pan pan.c && ./pan` (seguridad: deadlock,
   invalid end states, assertion violations; `-DNOCLAIM` ignora las
   fórmulas `ltl` para que sea un chequeo puro de seguridad)
3. `gcc -o pan pan.c && ./pan -a -N <ltl>` para cada una de
   `safe_update`, `termination`, `mutex`

Los resultados se guardan en `results/promela/` con nombres
`w<N>_j<M>_safety.txt` y `w<N>_j<M>_ltl_<nombre>.txt`.

### Alternativa con Docker

Es la forma en que se generaron los resultados de abajo (Windows sin
Spin/gcc instalados, con Docker Desktop):

```bash
docker run --rm -v "$PWD":/work -w /work debian:stable-slim bash -ec \
  'apt-get update -qq && apt-get install -y -qq spin gcc libc6-dev && \
   for v in "3 4" "2 4" "4 4"; do bash promela/run_spin.sh $v; done'
```

## Resultados de la verificación (Spin 6.5.2)

Ejecutado el 2026-09-28 con Spin 6.5.2 (Debian `stable-slim`, gcc).
Salidas completas en `results/promela/w<N>_j<M>_*.txt`.

| Variante (workers/jobs) | Corrida | Estados almacenados | Transiciones | Profundidad | Errores |
|---|---|---:|---:|---:|---:|
| 2/4 | seguridad (`-DSAFETY -DNOCLAIM`) | 12,358 | 18,374 | 114 | 0 |
| 2/4 | LTL `safe_update` | 12,358 | 18,375 | 220 | 0 |
| 2/4 | LTL `termination` | 12,311 | 48,834 | 217 | 0 |
| 2/4 | LTL `mutex` | 12,358 | 18,375 | 220 | 0 |
| 3/4 | seguridad | 145,563 | 244,504 | 123 | 0 |
| 3/4 | LTL `safe_update` | 145,563 | 244,505 | 235 | 0 |
| 3/4 | LTL `termination` | 145,218 | 632,301 | 232 | 0 |
| 3/4 | LTL `mutex` | 145,563 | 244,505 | 235 | 0 |
| 4/4 | seguridad | 1,080,202 | 1,967,768 | 132 | 0 |
| 4/4 | LTL `safe_update` | 1,080,202 | 1,967,769 | 250 | 0 |
| 4/4 | LTL `termination` | 1,078,409 | 5,003,428 | 247 | 0 |
| 4/4 | LTL `mutex` | 1,080,202 | 1,967,769 | 250 | 0 |

Conclusión: en las tres variantes, la búsqueda exhaustiva no encontró
violaciones de aserciones, estados finales inválidos (deadlock) ni
contraejemplos para `safe_update`, `termination` (verificada con
búsqueda de ciclos de aceptación, `-a`) y `mutex`. El modelo es libre
de condición de carrera sobre los pesos: solo el Coordinator entra a la
sección de actualización, y lo hace únicamente después de recibir los
`NUM_JOBS` resultados.

### Límite de explosión de estados

Se intentó también la variante 4/8. La corrida de seguridad superó
34 millones de estados almacenados y 3.2 GB de memoria sin terminar
(unos 214 s), así que se detuvo y se reemplazó por 4/4. El espacio de
estados crece de forma combinatoria con el número de jobs que están
en tránsito en los canales. Por eso la verificación exhaustiva se hace
con N pequeño: basta para cubrir todos los entrelazados de la lógica
de sincronización, que es independiente del tamaño real del dataset.

`verificacion_spin.txt` y `simulacion_spin.txt` se conservan como
evidencia de la corrida original de la PC2 (modelo previo a las
propiedades LTL; 143,799 estados y 0 errores).
