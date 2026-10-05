# Informe de GAPs: análisis de código con IA

Entregable del TP, punto (r). Resultado de aplicar el prompt documentado en
[`02-prompt-analisis-ia.md`](02-prompt-analisis-ia.md) al repositorio
`Grupo5-ProgramacionConcurrente`.

## 1. Resumen ejecutivo

El análisis no encontró defectos de severidad Alta. El núcleo concurrente
(`concurrent.go`) es correcto en los aspectos verificados: los pesos se
modifican únicamente después de `wg.Wait()`, la reducción es determinista por
`JobID` y el detector de carreras no reportó problemas. Los hallazgos se
concentran en robustez operativa (códigos de salida, escritura no atómica,
timeouts de red), cobertura de pruebas del pipeline de datos y deuda técnica
menor.

Total: **19 GAPs**.

| Severidad | Cantidad | GAPs |
|---|---|---|
| Alta | 0 | - |
| Media | 4 | GAP-01, GAP-02, GAP-03, GAP-04 |
| Baja | 15 | GAP-05 a GAP-19 |

| Categoría | Cantidad | GAPs |
|---|---|---|
| Seguridad | 3 | GAP-01, GAP-08, GAP-10 |
| Manejo de errores y robustez | 6 | GAP-02, GAP-03, GAP-07, GAP-09, GAP-11, GAP-12 |
| Pruebas y verificación | 1 | GAP-04 |
| Patrones de concurrencia y sincronización | 1 | GAP-05 |
| Rendimiento y memoria | 3 | GAP-06, GAP-16, GAP-17 |
| Calidad y mantenibilidad | 4 | GAP-13, GAP-14, GAP-15, GAP-19 |
| Reproducibilidad | 1 | GAP-18 |

Resultado de las hipótesis planteadas en el prompt:

| Hipótesis | Resultado | GAP |
|---|---|---|
| (a) Sin timeout de inactividad al transmitir el cuerpo de la descarga | Confirmada | GAP-01 |
| (b) Errores ignorados de `Flush`/`Error`/`Write`/`Close` en `report.go` | Parcialmente refutada: `Flush`/`Error`/`Write` sí se comprueban; solo se ignora el error de `Close` | GAP-12 |
| (c) Limpieza en `cpu_profile.go` ante fallos a mitad de ejecución | Mayormente refutada: no hay ruta de error entre `StartCPUProfile` y `StopCPUProfile`; quedan detalles menores | GAP-12 |
| (d) Claridad del dimensionamiento de buffers en `concurrent.go` | Confirmada como riesgo de mantenimiento: el código es correcto, pero depende de una invariante no documentada ni probada | GAP-05 |
| (e) Robustez de `run_spin.ps1` / `run_spin.sh` | Parcialmente confirmada (puntos menores) | GAP-18 |

## 2. Metodología

- **Prompt**: [`02-prompt-analisis-ia.md`](02-prompt-analisis-ia.md).
- **Modelo**: Claude (familia Claude 5), ejecutado mediante Claude Code.
- **Fecha**: 2026-10-03.
- **Commit analizado**: `cd142a2` (`git rev-parse --short HEAD`), rama
  `feature/tp-delivery`.
- **Lectura**: se leyeron completos todos los archivos Go no de prueba y los
  tres scripts de Spin. Las pruebas se revisaron a nivel de inventario
  (nombres de pruebas y funciones ejercitadas, obtenidos con búsquedas sobre
  `*_test.go`) y los fixtures de `equivalence_test.go` se leyeron completos.

Comandos ejecutados y resultados:

| Comando | Entorno | Resultado |
|---|---|---|
| `go vet ./...` | Local (Windows) | Sin salida: sin advertencias |
| `go test -count=1 ./...` | Local (Windows) | `ok  github.com/YuhiTTo/Grupo5-ProgramacionConcurrente  1.728s` |
| `go test -race -count=1 ./...` | Docker `golang:1.27` (Go 1.27.1 linux/amd64, `CGO_ENABLED=1`), 2026-10-03 | `ok ... 1.448s`. `equivalence_test.go` ejercita `trainConcurrent` con 1, 2, 4 y 8 workers, por lo que el detector cubrió el worker pool |
| `gofmt -l .` | Local (Windows) | Lista todos los archivos porque la copia de trabajo usa fin de línea CRLF (el índice de git es LF). Normalizando con `tr -d '\r'`, solo `preprocessing.go` queda sin formatear (GAP-19) |
| Spin (`promela/run_spin.sh`, `run_spin_mutants.sh`) | Evidencia previa del equipo en `results/promela/` y `results/promela/mutants/` | `safe_update`, `termination` y `mutex` con `errors: 0` para 2/4, 3/4 y 4/4 workers/jobs; los modelos mutantes produjeron `errors: 1` en las tres comprobaciones realizadas |

Las pruebas pasan: el análisis no se basa en suposiciones sobre un estado
roto del repositorio. Los comandos de Spin no se volvieron a ejecutar en este
análisis; se reportan como evidencia existente.

Convención: cada GAP cita `archivo:línea` verificado sobre el commit
`cd142a2`. Lo que no se pudo comprobar con una ejecución se marca como
**hipótesis**.

## 3. Fortalezas

Aspectos revisados que son correctos y no constituyen GAPs:

1. **Sincronización del worker pool.** Los workers solo leen `model`; los pesos
   se escriben después de recibir todos los resultados y de `wg.Wait()`
   (`concurrent.go:157-165`, `194-205`). No hay estado mutable compartido
   durante la fase paralela. El detector de carreras pasó con 1, 2, 4 y 8
   workers.
2. **Reducción determinista.** Los resultados se indexan por `JobID`
   (`concurrent.go:154-163`) y se suman siempre en el mismo orden
   (`173-189`). `TestConcurrentTrainingIsDeterministic`
   (`equivalence_test.go:79`) exige igualdad exacta entre dos corridas.
3. **Disciplina de canales.** El productor cierra `jobs` (`concurrent.go:148`);
   los workers drenan con `range` y llaman `defer wg.Done()` (`28-30`). No se
   observan fugas de goroutines: todas terminan al agotarse `jobs`.
4. **Descarga segura.** HTTPS obligatorio y allowlist de hosts validados
   también en cada redirección (`dataset_download.go:178`, `188-194`, `227-242`),
   límite de redirecciones (`40`), rechazo de HTML (`206`), contraste de
   tamaño anunciado (`214`), `io.LimitReader(Size+1)` (`301`), escritura a
   `.part` con SHA-256 al vuelo, `Sync`, y `Rename` atómico (`274-341`).
5. **Validación de la CLI.** `-workers` rechaza valores no numéricos o `< 1` y
   listas vacías (`config.go:184-199`); `validate` exige `runs >= 1`,
   `warmup >= 0`, `epochs >= 1` y `trim` en `[0, 0.5)` (`config.go:229-243`),
   con pruebas (`config_test.go:109-160`).
6. **Divisiones protegidas en el núcleo numérico.** Desviación estándar nula
   (`scaling.go:81-100`), conjunto vacío (`scaling.go:23`,
   `regression.go:108`, `concurrent.go:69`, `sequential.go:13`), media nula en
   el coeficiente de variación (`stats.go:65`) y varianza total nula en R²
   (`regression.go:161`).
7. **Ausencia de fuga de datos.** Los parámetros de escalamiento se calculan
   solo con Train (`main.go:283`) y la partición usa semilla fija
   (`main.go:244`).
8. **Escritura de reportes.** `csv.Writer` se vacía y se consulta
   `writer.Error()` (`report.go:85-89`, `163-167`, `301-305`); los errores de
   persistencia del benchmark se reportan sin abortar la medición
   (`main.go:701-762`).
9. **Instrumentación sincronizada.** El muestreador de recursos escribe
   `peakHeap`/`peakGoroutines` y el hilo principal los lee solo tras
   `wg.Wait()` (`resource_profile.go:78-79`).
10. **Envoltura de errores.** Uso consistente de `%w` (por ejemplo
    `dataset_download.go:134`, `146`, `153`, `dataset.go:19-22`).
11. **Verificación formal con mutantes.** Los mutantes comprueban que la
    verificación no es vacua, y el script falla si un mutante no se detecta
    (`run_spin_mutants.sh:49-60`, `103-106`).
12. **Pruebas de la descarga.** Servidor `httptest`, hash incorrecto, cuerpo
    truncado, redirección a host no permitido, cancelación de contexto
    (`dataset_download_test.go:88-315`).
13. **Reproducibilidad.** `environment.md` registra versión de Go, plataforma,
    CPUs, `GOMAXPROCS` y parámetros (`report.go:213-254`).

## 4. Tabla de GAPs

| ID | Categoría | Severidad | Evidencia | Descripción | Impacto | Recomendación | Estado |
|---|---|---|---|---|---|---|---|
| GAP-01 | Seguridad | Media | `dataset_download.go:82-92`, `303`; `main.go:64`, `147` | La descarga solo tiene timeouts de conexión/cabeceras; el cuerpo se lee sin límite de inactividad y con `context.Background()` | Una conexión detenida deja el CLI colgado indefinidamente (la descarga es automática en el modo por defecto) | Envolver `resp.Body` en un lector con timeout de inactividad (o contexto con `WithCancel` y temporizador reiniciable) | Corregido (`e171945`) |
| GAP-02 | Manejo de errores y robustez | Media | `main.go:167-170`, `180-183`, `226-229`, `248-251`, `286-289`; `cleaning.go:75-78` y siguientes; `main.go:43` | `runPipeline` y `runCleaning` imprimen "Error" en stdout y retornan: el proceso termina con código 0 | Scripts o CI no detectan el fallo; inconsistente con otras rutas que sí hacen `os.Exit(1)` (`main.go:25`, `30`, `336`, `372`) | Hacer que ambas funciones devuelvan `error` y que `main` salga con código distinto de 0; escribir errores a stderr | Corregido (`c7d3017`) |
| GAP-03 | Manejo de errores y robustez | Media | `cleaning.go:97-105`, `121-124`, `143-146`; `main.go:127` | El dataset limpio se escribe directamente en su ruta final; ante un error a mitad de proceso `defer csvWriter.Flush()` deja un archivo truncado que `ensureNamedDatasetAvailable` considera válido por existir | El pipeline puede entrenar con un CSV incompleto sin advertirlo | Escribir a `.part` y renombrar al finalizar (como `streamToFile`), y devolver el error | Corregido (`411ebd5`) |
| GAP-04 | Pruebas y verificación | Media | `equivalence_test.go:29-57`; ver sección 5 | Sin pruebas para `cleaning.go`, `preprocessing.go`, `scaling.go`, `regression.go`, `dataset.go`, `cpu_profile.go` y `runPipeline`; el fixture de equivalencia solo usa features directas (categóricas en `-1`) | Errores de limpieza, indexación one-hot o gradiente categórico no serían detectados por la suite | Agregar pruebas unitarias de parseo/limpieza, esquema one-hot, escalamiento y un fixture con dummies activas; casos borde de `trainConcurrent` | Corregido (`dafc6bb`) |
| GAP-05 | Patrones de concurrencia y sincronización | Baja | `concurrent.go:82-86`, `104-108`, `125-146`, `157-163` | La ausencia de deadlock depende de la invariante `actualJobCount <= jobCount` (capacidad de ambos canales); no está documentada ni probada | Un cambio futuro en el cálculo de `chunkSize` podría producir un deadlock difícil de diagnosticar | Documentar la invariante, agregar una prueba o `panic` defensivo, o consumir `results` en una goroutine aparte | Corregido (`b608587`) |
| GAP-06 | Rendimiento y memoria | Baja | `concurrent.go:102-123`, `32-33`, `154-155`, `167-168` | En cada época se crean 2 canales, `workerCount` goroutines y varios slices (gradiente local por job, `partialResults`, `totalGradients`) | Sobrecarga constante por época que limita el speedup con muchos workers (hipótesis; no medida con `pprof` en este análisis) | Pool persistente de workers y buffers de gradiente reutilizables | Pendiente |
| GAP-07 | Manejo de errores y robustez | Baja | `concurrent.go:191-205`, `sequential.go:63-77`, `main.go:326`, `526-527`, `657-659` | No hay detección de valores no finitos (divergencia con `learningRate` alto); los speedups dividen duraciones sin guarda de cero | Con otro `learningRate` o datos sin escalar se obtendrían `NaN`/`Inf` sin aviso (hipótesis; con 0.01 y datos escalados converge) | Verificar `math.IsNaN`/`IsInf` en la MSE por época y proteger las divisiones | Pendiente |
| GAP-08 | Seguridad | Baja | `dataset_download.go:83-89` | El `http.Transport` se crea sin `Proxy`, por lo que ignora `HTTP(S)_PROXY`; además, al fijar `DialContext` HTTP/2 queda desactivado (documentación de `net/http`, no ejecutado aquí) | La descarga falla o se degrada en redes con proxy obligatorio | Usar `Proxy: http.ProxyFromEnvironment` y `ForceAttemptHTTP2: true` | Pendiente |
| GAP-09 | Manejo de errores y robustez | Baja | `main.go:64`; `dataset_download.go:283-292`, `303` | Sin manejo de señales ni reanudación: ante Ctrl-C queda un `.part` (solo se elimina en rutas de error); no hay reintentos | Un corte reinicia la descarga de hasta 947 MB; archivo residual en disco | `signal.NotifyContext` para cancelar y limpiar; reintento acotado o `Range` | Pendiente |
| GAP-10 | Seguridad | Baja | `main.go:101-112`, `127`; `dataset_download.go:106-107` | El pipeline solo comprueba la existencia del archivo; el tamaño y SHA-256 se verifican únicamente dentro de `ensureDataset` (modo `download`) | Un dataset corrupto o desactualizado en la ruta por defecto se usa sin verificación | Verificar tamaño al menos (barato) y ofrecer una opción para el hash completo | Pendiente |
| GAP-11 | Manejo de errores y robustez | Baja | `preprocessing.go:34`, `67-72`, `308`, `391` | Las lecturas usan `FieldsPerRecord = -1` y `row[columnIndex[nombre]]`: una columna ausente devuelve el índice 0 y una fila corta provoca `panic` por índice fuera de rango; no se validan columnas requeridas como en `cleaning.go:92` | Con un `-dataset` ajeno, `panic` con traza o lectura de una columna incorrecta | Validar columnas obligatorias con `findMissingRequiredColumn` y la longitud de cada fila | Corregido (`a6e1086`) |
| GAP-12 | Manejo de errores y robustez | Baja | `report.go:59`, `149`, `271`; `cpu_profile.go:47-51`, `68`, `73`; `cleaning.go:102` | El error de `Close` de archivos escritos se ignora (`defer file.Close()`); `StopCPUProfile` no está en `defer`; se informa "Perfil guardado" sin comprobar el cierre | Un fallo de escritura diferida (disco lleno) podría pasar inadvertido; sin impacto en el flujo normal | Cerrar con comprobación de error (patrón `defer` con error nombrado) y `defer pprof.StopCPUProfile()` | Corregido (`6f526f9`) |
| GAP-13 | Calidad y mantenibilidad | Baja | `main.go:244`, `326`, `464`, `573`, `771`; `cpu_profile.go:22`, `24`; `cleaning.go:14` | Valores fijos: partición 0.80, semilla 42, `learningRate` 0.01, tolerancia 1e-9, `resourceRuns` 3, `profileEpochs` 3000; `cpu-profile` y `quick` ignoran `-epochs`/`-workers` y usan `NumCPU`; el nombre del CSV crudo incluye una fecha | Parámetros no configurables; flags que parecen aplicar y no lo hacen | Exponer como flags o constantes nombradas y documentar qué modos ignoran qué flags | Pendiente |
| GAP-14 | Calidad y mantenibilidad | Baja | `regression.go:28-32`, `73-77`; `preprocessing.go:159` | Los índices 0 a 4 de los pesos están escritos a mano y acoplados a `directFeatureCount = 5` y al orden de `Sample` | `predict` entra en `panic` si `len(Weights) < 5`; riesgo al agregar features | Constantes con nombre para los índices y verificación en `newLinearRegression` | Pendiente |
| GAP-15 | Calidad y mantenibilidad | Baja | `benchmark.go:28-92` y `94-162`; `resource_profile.go:112-251`; `sequential.go:63-79` y `concurrent.go:191-209`; `main.go:395-596` | Código duplicado entre variantes secuencial/concurrente (benchmark, perfil de recursos, actualización de pesos) y funciones largas en `main.go` (857 líneas) | Cambios deben replicarse en varios sitios | Extraer funciones comunes (`applyGradient`, medición genérica recibiendo `func()`) | Pendiente |
| GAP-16 | Rendimiento y memoria | Baja | `dataset.go:41`, `preprocessing.go:53`, `344`; `main.go:165-246` | El dataset limpio se recorre tres veces (conteo, esquema, carga) | Tiempo de arranque adicional con 2.1 M de filas | Fusionar conteo y esquema en una sola pasada | Pendiente |
| GAP-17 | Rendimiento y memoria | Baja | `resource_profile.go:47-69` | El muestreador llama a `runtime.ReadMemStats` (detiene el mundo) cada 10 ms durante la corrida medida; el pico puede omitir picos más cortos | `elapsed_ms` de `resources.csv` queda perturbado (hipótesis; efecto no medido) y el pico es una cota inferior | Documentar que `elapsed_ms` no es comparable con el benchmark, o subir el período | Pendiente |
| GAP-18 | Reproducibilidad | Baja | `promela/run_spin.sh:8-11`, `28-29`, `34`, `78`; `promela/run_spin.ps1:9-12`, `35`, `72`; `run_spin_mutants.sh:83`, `97`, `54-58` | `run_spin.sh` crea el directorio temporal sin `trap` (queda si falla), no valida que `spin`/`gcc` existan ni que los parámetros sean enteros; comentarios obsoletos ("NO están instalados"); `.ps1` usa `\` y `pan.exe` (solo Windows) y no hay versión PowerShell de los mutantes; el chequeo de mutantes no distingue el tipo de error | Residuos en `/tmp`, mensajes de error poco claros y reproducción menos portable | `trap 'rm -rf "$WORK_DIR"' EXIT` y `command -v` previo (como ya hace `run_spin_mutants.sh:40`); actualizar comentarios | Parcial (`21bf818`): `run_spin.sh` incorpora limpieza mediante `trap`, validación de argumentos y comprobación de `spin`/`gcc`; `run_spin.ps1` controla errores y limpia recursos, pero aún no replica la validación explícita de argumentos positivos ni la comprobación previa de dependencias, y no existe una versión PowerShell del script de mutantes |
| GAP-19 | Calidad y mantenibilidad | Baja | `preprocessing.go:13-21`; `config.go:10`; repositorio sin `.github/workflows` | `PreprocessingSummary` no está formateado con `gofmt`; typo "quando" en un comentario; no hay integración continua que ejecute `go vet`, `go test -race` y `gofmt` | Deriva de estilo y regresiones no detectadas automáticamente | `gofmt -w`, corregir el typo y agregar un workflow de GitHub Actions | Parcial (`91fcea8`, `1142d86`): `gofmt` y typo corregidos (verificado sobre el contenido versionado); CI pendiente |

## 5. Detalle por GAP

### GAP-01: sin timeout de inactividad al transmitir el cuerpo (Media)

Hecho verificado: el cliente define timeouts para conexión, handshake TLS y
cabeceras, y omite deliberadamente `Client.Timeout` para no cortar descargas
largas (`dataset_download.go:78-92`). Una vez recibidas las cabeceras, el
cuerpo se copia con `io.Copy` sin ningún límite de inactividad, y el contexto
es `context.Background()` sin plazo ni cancelación (`main.go:64`, `147`):

```go
// dataset_download.go:301-303
limited := io.LimitReader(body, spec.Size+1)
written, err := io.Copy(multi, limited)
```

La prueba de cancelación (`dataset_download_test.go:287`) usa un contexto ya
cancelado, no una conexión detenida. **Hipótesis**: un servidor que deja de
enviar datos sin cerrar la conexión bloquearía la lectura indefinidamente
(no se simuló en este análisis).

### GAP-02: código de salida 0 ante errores del pipeline (Media)

`runPipeline` no devuelve `error`. Cada fallo imprime y retorna:

```go
// main.go:167-170
if err != nil {
    fmt.Println("Error:", err)
    return
}
```

Lo mismo ocurre en `runCleaning` (`cleaning.go:75-78`, `85-88`, `93-95`,
`98-101`, `108-110`, `121-124`, `144-146`, `153-156`), invocada en
`main.go:43`. En cambio, `main.go:25`, `30` y `39` sí salen con 1, por lo que
el comportamiento es inconsistente.

### GAP-03: escritura no atómica del dataset limpio (Media)

```go
// cleaning.go:97-105
outputFile, err := os.Create(outputCSVPath)
...
defer outputFile.Close()
csvWriter := csv.NewWriter(outputFile)
defer csvWriter.Flush()
```

Si falla la lectura de una fila (`121-124`) o la escritura (`143-146`), la
función retorna y el `Flush` diferido deja un CSV parcial en la ruta final.
Como `ensureNamedDatasetAvailable` solo comprueba existencia (`main.go:127`),
una ejecución posterior lo usaría. Contrasta con `streamToFile`
(`dataset_download.go:274-341`), que ya implementa el patrón `.part` +
`Rename`.

### GAP-04: pruebas ausentes para el pipeline de datos (Media)

Hecho verificado con búsquedas en `*_test.go`: no hay referencias a
`parseCurrency`, `parseLengthOfStay`, `cleanRecord`, `buildFeatureSchema`,
`analyzePreprocessingSchema`, `loadAndSplitDataset`, `parseSample`,
`calculateScalingParameters`, `evaluateModel`, `restoreTargetScale`,
`inspectCleanDataset`, `runCPUProfile` ni `runPipeline`. La suite tiene 59
pruebas en 7 archivos, concentradas en descarga (14), configuración (15),
reportes (11), estadísticas (8), `main` (6), equivalencia (4) y recursos (1).

Además, el fixture de equivalencia fija todas las categóricas en `-1`
(`equivalence_test.go:46-50`), con 64 muestras, 5 features y 25 épocas, de
modo que las ramas `AgeFeature >= 0` y similares de
`accumulateSampleGradient` (`regression.go:79-97`) no se ejercitan en el
camino concurrente. Tampoco hay casos borde para `trainConcurrent` (datos
vacíos, `workers` mayor que las muestras, `jobCount` recortado en
`concurrent.go:84-86`). Mitigación existente: la equivalencia a escala
completa está documentada en el README (`README.md:81`).

### GAP-05: invariante implícita en los buffers de canales (Baja)

```go
// concurrent.go:82-86, 104-108
jobCount := workerCount * 4
if jobCount > len(trainData) { jobCount = len(trainData) }
jobs := make(chan GradientJob, jobCount)
results := make(chan GradientResult, jobCount)
```

Con `chunkSize = ceil(n/jobCount)` (`125-127`), el número de jobs generados
es `ceil(n/chunkSize) <= jobCount`, de modo que el productor nunca se bloquea
al enviar (`139`) y los workers nunca se bloquean al publicar (`51`).
Esto es correcto y se verificó por razonamiento aritmético. El riesgo es que
`results` solo se consume después de terminar de producir (`157-163`): si
algún cambio hiciera `actualJobCount > jobCount`, el productor se bloquearía
con `jobs` lleno mientras los workers se bloquean con `results` lleno
(deadlock). **Hipótesis**: ese escenario no es alcanzable hoy; el modelo
Promela verifica la estructura general, no esta aritmética concreta.

### GAP-06: goroutines y asignaciones por época (Baja)

`trainConcurrent` crea dentro del bucle de épocas (`concurrent.go:102-123`)
dos canales y `workerCount` goroutines, y cada job reserva su vector de
gradiente (`32-33`). Con 32 workers y 100 épocas se lanzan 3 200 goroutines
por entrenamiento. Es una decisión de diseño legítima que mantiene el
código simple y verificable, pero añade costo fijo por época. No se midió su
peso con `pprof`; es una hipótesis de rendimiento.

### GAP-07: sin detección de divergencia ni guarda de división (Baja)

La MSE por época se calcula (`concurrent.go:207-209`) pero no se comprueba
que sea finita. `learningRate` es constante en `main.go:326` y no es un flag,
por lo que hoy no es explotable desde la CLI. En `main.go:526-527` y
`657-659` se dividen duraciones sin guarda de cero; en la práctica una
duración exactamente cero es improbable.

### GAP-08: `Transport` sin proxy ni HTTP/2 (Baja)

```go
// dataset_download.go:83-89
transport := &http.Transport{
    DialContext: (&net.Dialer{Timeout: 30 * time.Second}).DialContext,
    TLSHandshakeTimeout:   30 * time.Second,
    ResponseHeaderTimeout: 30 * time.Second,
}
```

Un `Transport` literal tiene `Proxy` nulo, es decir, no usa las variables de
entorno de proxy. Según la documentación de `net/http`, definir `DialContext`
también desactiva HTTP/2 salvo `ForceAttemptHTTP2`. No se probó en una red
con proxy.

### GAP-09: sin manejo de señales ni reanudación (Baja)

`abort` (`dataset_download.go:288-292`) borra el `.part` solo cuando
`streamToFile` detecta un error. Una interrupción del proceso no ejecuta esa
ruta y deja el archivo parcial; la siguiente descarga lo trunca con
`os.Create` (`283`), de modo que no hay corrupción, pero tampoco reanudación
ni reintentos.

### GAP-10: dataset existente sin verificación en el pipeline (Baja)

`ensureCleanDatasetAvailable` y `ensureNamedDatasetAvailable` usan
`fileExists` (`main.go:101-112`, `127`). La verificación de tamaño y hash
existe (`dataset_download.go:131-166`) pero solo se invoca desde
`ensureDataset`. Hay un compromiso de costo: calcular SHA-256 de 244 MB en
cada ejecución tiene un costo medible, por eso se sugiere al menos validar el
tamaño.

### GAP-11: acceso por nombre de columna sin validación (Baja)

```go
// preprocessing.go:67-68
ageGroups[row[columnIndex["Age Group"]]] = struct{}{}
admissionTypes[row[columnIndex["Type of Admission"]]] = struct{}{}
```

Un mapa devuelve el valor cero para claves inexistentes, así que una
columna ausente se lee como la columna 0; con `FieldsPerRecord = -1`
(`34`, `308`) una fila corta provoca `panic`. Con el CSV generado por el
propio proyecto no ocurre; el riesgo aparece con `-dataset` externo.

### GAP-12: errores de `Close` y limpieza del perfil de CPU (Baja)

Hipótesis (b): `Flush` y `Error` sí se comprueban en los tres escritores CSV
(`report.go:85-89`, `163-167`, `301-305`) y `os.WriteFile` devuelve su error;
solo se ignora el error de `Close` diferido (`59`, `149`, `271`).
Hipótesis (c): entre `pprof.StartCPUProfile` (`cpu_profile.go:49`) y
`pprof.StopCPUProfile` (`68`) no existe ninguna ruta de retorno de error;
si `trainConcurrent` entrara en `panic`, el proceso termina de todos modos.
El único hallazgo es de higiene: `Stop` no está en `defer` y el mensaje de
éxito se imprime sin comprobar el cierre del archivo.

### GAP-13: valores fijos y flags que se ignoran (Baja)

`runCPUProfile` usa `runtime.NumCPU()` y 3 000 épocas fijas
(`cpu_profile.go:22-24`), ignorando `-workers` y `-epochs`; el modo `quick`
también usa `NumCPU` (`main.go:464`). La tolerancia de equivalencia, la
semilla, la proporción de partición y la tasa de aprendizaje son constantes
locales.

### GAP-14: acoplamiento por índices fijos en `predict` (Baja)

`predict` y `accumulateSampleGradient` acceden a `Weights[0]`..`Weights[4]`
(`regression.go:28-32`, `73-77`); ese 5 equivale a `directFeatureCount`
(`preprocessing.go:159`) y al orden de los campos de `Sample`. No hay
constantes con nombre ni verificación de tamaño.

### GAP-15: duplicación y funciones largas (Baja)

`benchmarkSequential` y `benchmarkConcurrent` difieren solo en la función de
entrenamiento; ocurre lo mismo en `profileSequentialResources` y
`profileConcurrentResources`. La actualización de pesos está duplicada en
`sequential.go:63-79` y `concurrent.go:191-209`. `runQuickComparison`
supera 200 líneas.

### GAP-16: tres pasadas sobre el CSV (Baja)

`inspectCleanDataset` (`dataset.go:41`), `analyzePreprocessingSchema`
(`preprocessing.go:53`) y `loadAndSplitDataset` (`preprocessing.go:344`)
leen todo el archivo. La primera solo cuenta filas para dimensionar los
slices.

### GAP-17: sobrecarga del muestreador de recursos (Baja)

`runtime.ReadMemStats` detiene el mundo; ejecutarlo cada 10 ms
(`resource_profile.go:47-56`) durante la corrida puede afectar el tiempo
medido. El modo `resources` es independiente del benchmark formal, por lo
que el impacto se limita a `elapsed_ms` de `resources.csv`.

### GAP-18: robustez de los scripts de Spin (Baja)

Hipótesis (e), puntos confirmados: `run_spin.sh` no tiene `trap` de limpieza
(`34` frente a la limpieza solo al final, `78`), mientras que
`run_spin_mutants.sh:40` sí; no verifica herramientas ni valida parámetros
(`28-29`); los comentarios iniciales afirman que `spin` y `gcc` "NO están
instalados" (`run_spin.sh:8-11`, `run_spin.ps1:9-12`), lo que ya no describe
el estado del proyecto. Puntos correctos: `set -euo pipefail`, comprobación
explícita de `errors: 0`, revisión de `$LASTEXITCODE` y `finally` en el
`.ps1`.

### GAP-19: formato, typo y ausencia de CI (Baja)

`gofmt -d` sobre `preprocessing.go` (normalizado a LF) muestra la alineación
incorrecta de los campos de `PreprocessingSummary` (`13-21`). Typo en
`config.go:10`. `git ls-files` no lista flujos de trabajo ni `Makefile`.

## 6. Priorización

| Prioridad | GAPs | Criterio | Esfuerzo estimado |
|---|---|---|---|
| 1 | GAP-02, GAP-03 | Resultados silenciosamente inválidos: salida 0 y archivos truncados reutilizados | Bajo (propagar `error`, `.part` + `Rename`) |
| 2 | GAP-01 | Cuelgue indefinido en la descarga automática | Bajo a medio |
| 3 | GAP-04 | Cobertura del pipeline que alimenta al algoritmo concurrente | Medio |
| 4 | GAP-05, GAP-10, GAP-11 | Fragilidad latente ante cambios o entradas externas | Bajo |
| 5 | GAP-08, GAP-09, GAP-12, GAP-18 | Portabilidad y higiene operativa | Bajo |
| 6 | GAP-06, GAP-07, GAP-13 a GAP-17, GAP-19 | Deuda técnica y optimizaciones | Bajo a medio |

Para el objetivo académico del TP (comparar secuencial y concurrente) ningún
GAP invalida los resultados del benchmark; los de prioridad 1 y 3 afectan la
confianza en el pipeline de datos, no el núcleo concurrente.

## 7. Limitaciones

- Análisis estático asistido por IA, sin ejecución del pipeline completo
  sobre el dataset de 947 MB; no se midió rendimiento ni se hicieron perfiles.
- Las pruebas se revisaron por inventario, no se leyeron todas línea a línea;
  la ausencia de pruebas se determinó por búsqueda textual de símbolos en
  `*_test.go`.
- Los archivos `promela/*.pml` no se analizaron en detalle: se evaluaron los
  scripts y los resultados guardados. La verificación formal valida el modelo
  abstracto, no el código Go; la correspondencia modelo-código no se comprobó.
- El detector de carreras se ejecutó en Docker; en la máquina local de
  desarrollo no se repitió. La cobertura del detector depende de los
  escenarios de las pruebas (64 muestras, 1 a 8 workers).
- Las afirmaciones sobre `net/http` (proxy, HTTP/2) provienen de la
  documentación de la biblioteca estándar y no se reprodujeron.
- Las severidades son un juicio del analizador según las definiciones del
  prompt y deben ser revisadas por el equipo.
- El modelo de IA puede equivocarse: las citas `archivo:línea` se
  verificaron contra el commit `cd142a2`, pero cambios posteriores pueden
  desplazar las líneas.

## 8. Corrección de los GAPs

Después del análisis, el equipo corrigió los 4 GAPs de severidad media y aplicó mejoras sobre 5 GAPs de severidad baja. La revisión posterior confirmó que GAP-05, GAP-11 y GAP-12 se encuentran corregidos, mientras que GAP-18 y GAP-19 permanecen parcialmente corregidos. Los GAPs restantes (06, 07, 08, 09, 10, 13, 14, 15, 16 y 17) quedan como mejoras futuras: son de severidad baja y no invalidan las propiedades de concurrencia verificadas para el Worker Pool.

| GAP | Corrección | Prueba que la respalda |
|---|---|---|
| GAP-01 | `idleTimeoutReader` cancela la descarga si no llegan bytes durante `datasetIdleTimeout` (60 s) y elimina el `.part` | Servidor `httptest` que envía datos y se detiene |
| GAP-02 | `runPipeline` y `runCleaning` devuelven `error`; `main` lo escribe en stderr y sale con código 1 | Dataset inexistente devuelve error (verificado también: `go run . -mode=quick -no-download -dataset nonexistent.csv` sale con 1) |
| GAP-03 | El CSV limpio se escribe en `.part` y se renombra al terminar; ante error se elimina el `.part` | El fallo no deja archivo final y no modifica uno previo |
| GAP-04 | Pruebas de preprocesamiento, escalamiento, regresión y equivalencia con dummies categóricas activas (1, 2, 4 y 8 workers) | Las propias pruebas; una mutación de la rama categórica del gradiente hace fallar la suite |
| GAP-05 | Invariante `actualJobCount <= jobCount` documentada | Casos borde de particionamiento (datos que no dividen exacto y menos datos que `workers × 4`) |
| GAP-11 | Validación de columnas obligatorias y longitud de filas | Columna ausente y fila corta devuelven error en lugar de `panic` |
| GAP-12 | Cierre de archivos con error comprobado; `defer pprof.StopCPUProfile()` | Prueba del helper `closeFile` |
| GAP-18 | `run_spin.sh`: `trap` de limpieza, verificación de `spin`/`gcc` y validación de argumentos | Ejecución en Docker y casos de error manuales (argumento no numérico, `spin` ausente) |
| GAP-19 | `gofmt` y corrección del typo; falta el workflow de CI | `git show HEAD:preprocessing.go \| gofmt -l` no reporta nada |

### Revisión crítica posterior de los GAPs de severidad media

Como parte de la revisión final del TP, se contrastaron los cuatro GAPs de severidad media identificados por la IA con el estado actual del código y de la suite de pruebas.

- **GAP-01 — Timeout de inactividad en la descarga:** se verificó que `dataset_download.go` incorpora `datasetIdleTimeout` y el lector `idleTimeoutReader`, que cancela la operación cuando el servidor deja de enviar datos durante el intervalo configurado. Además, `dataset_download_test.go` contiene una prueba específica para este escenario. Por ello, se mantiene el estado **Corregido**.

- **GAP-02 — Propagación de errores y código de salida:** se comprobó que `runPipeline` y `runCleaning` retornan errores al llamador y que `main()` los reporta mediante `stderr` y finaliza con código distinto de cero mediante `os.Exit(1)`. Por ello, se mantiene el estado **Corregido**.

- **GAP-03 — Escritura atómica del dataset limpio:** se verificó que la limpieza escribe primero en un archivo temporal `.part` y solo reemplaza la salida final mediante `os.Rename` cuando el procesamiento termina correctamente. Las pruebas comprueban tanto la eliminación del archivo temporal ante errores como la conservación de una salida previa. Por ello, se mantiene el estado **Corregido**.

- **GAP-04 — Cobertura insuficiente de pruebas:** se comprobó la incorporación de pruebas para limpieza, preprocesamiento, escalamiento, regresión y ejecución del pipeline, además de casos de equivalencia con variables categóricas activas y diferentes cantidades de workers. Por ello, se mantiene el estado **Corregido**.

La revisión confirma que los cuatro hallazgos de severidad media ya no permanecen abiertos en el estado actual del repositorio. No obstante, esta validación no implica que el software esté libre de defectos; únicamente confirma que las condiciones específicas señaladas originalmente por estos GAPs cuentan actualmente con una corrección y evidencia de prueba asociada.

### Revisión de GAPs de severidad baja y estado residual

Como segunda etapa de la revisión final, se contrastaron con el código actual los cinco GAPs de severidad baja sobre los que el equipo había aplicado correcciones o mejoras.

- **GAP-05 — Invariante del Worker Pool:** se verificó que `concurrent.go` documenta explícitamente la relación `actualJobCount <= jobCount` y explica su dependencia del cálculo de `chunkSize`. Además, `equivalence_test.go` incorpora casos borde con tamaños no divisibles, menos muestras que `workers × 4` y más workers que muestras. Por ello, se considera **Corregido** para la implementación actual.

- **GAP-11 — Validación de estructura del dataset:** se comprobó la validación de columnas obligatorias y de la longitud mínima de las filas antes de su procesamiento. Asimismo, existen pruebas específicas para columnas ausentes y filas incompletas. Por ello, se considera **Corregido**.

- **GAP-12 — Manejo de errores de cierre y perfilado de CPU:** las funciones de reporte utilizan `closeFile` para propagar errores de `Close`, comportamiento que cuenta con pruebas específicas. El perfilado de CPU incorpora limpieza diferida mediante `defer pprof.StopCPUProfile()` y comprueba explícitamente el cierre del archivo en la ruta normal. Por ello, se considera **Corregido**, aunque se mantiene como observación menor la redundancia entre la limpieza explícita y diferida.

- **GAP-18 — Robustez de los scripts de Spin:** `run_spin.sh` valida argumentos, comprueba la disponibilidad de `spin` y `gcc`, limpia recursos temporales mediante `trap` y verifica explícitamente `errors: 0`. `run_spin.ps1` también controla errores de procesos nativos, verifica los resultados y elimina el directorio temporal mediante `try/finally`; sin embargo, no replica todavía la validación explícita de argumentos positivos ni la comprobación previa de dependencias. Por ello, se mantiene como **Parcialmente corregido**.

- **GAP-19 — Formato y mantenibilidad:** se aplicaron `gofmt` y la corrección del error tipográfico. Ejecutar `gofmt -l preprocessing.go` en la copia de trabajo de Windows reporta el archivo, pero es un falso positivo: Git convierte los finales de línea a CRLF al hacer checkout (ver la sección 2). Sobre el contenido versionado, `git show HEAD:preprocessing.go | gofmt -l` no reporta nada, y lo mismo ocurre con todos los archivos `.go`. Se mantiene como **Parcialmente corregido** solo porque falta el workflow de integración continua.

La revisión posterior muestra que las mejoras aplicadas no eliminan automáticamente todos los hallazgos identificados por la IA. Tres de los cinco GAPs de severidad baja revisados cuentan con evidencia suficiente para considerarse corregidos, mientras que GAP-18 y GAP-19 conservan aspectos menores pendientes. Estos resultados permiten mantener en el informe una clasificación acorde con el estado real del repositorio.

### Verificación posterior a las correcciones

| Comando | Resultado |
|---|---|
| `go vet ./...` | Sin observaciones |
| `go test -count=1 ./...` | `ok` |
| `go test -race -count=1 ./...` (Docker, `golang:1.27`) | `ok`, sin carreras (evidencia en `results/race/`) |
| `go run . -mode=quick` | Equivalencia secuencial/concurrente: OK |
| `bash promela/run_spin.sh 2 4`, `3 4` y `4 4`; `bash promela/run_spin_mutants.sh` | Modelo correcto con `errors: 0` en las tres configuraciones; las 3 comprobaciones sobre los modelos mutantes detectaron correctamente los defectos introducidos con `errors: 1` |

