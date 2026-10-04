# 05 — Guion del video (TP, punto v)

Requisito: video de **máximo 6 minutos** en el que **cada integrante** demuestre
conocimiento del tema y presente los resultados, con una discusión crítica
sobre limitaciones, escalabilidad y posibles mejoras. El enlace se publica en
la nube y va como anexo del informe. Si falta el video, se descuentan 5 puntos.

## Reparto

El equipo completa esta tabla antes de grabar:

| Bloque | Tiempo | Tema | Integrante |
|---|---|---|---|
| 1 | 0:00–1:30 | Presentación, problema y dataset | Integrante 1: ________ |
| 2 | 1:30–3:30 | Worker Pool y verificación formal en Spin | Integrante 2: ________ |
| 3 | 3:30–5:15 | Resultados de rendimiento y análisis con IA | Integrante 3: ________ |
| 4 | 5:15–6:00 | Discusión crítica (los tres) | Todos |

Total: 6:00. Conviene apuntar a 5:40 para tener margen.

## Preparación antes de grabar

1. Clonar o actualizar el repositorio (`git pull` en `main`) y descargar el
   dataset una vez: `go run . -mode=download`.
2. Grabar aparte la salida de `go run . -mode=quick` (tarda casi un minuto) para
   insertarla acelerada en la edición, o tenerla ya ejecutada en una terminal.
3. No instalar Spin en vivo. Mostrar los resultados ya guardados en
   `results/promela/` y `results/promela/mutants/`.
4. Editor y terminal con letra grande (al menos 18 pt) y tema claro.
5. Dejar abiertas, en este orden:
   1. `README.md` en GitHub.
   2. `concurrent.go`.
   3. `promela/regression_workers.pml`.
   4. `results/promela/w3_j4_ltl_mutex.txt`.
   5. `docs/tp/01-verificacion-formal.md`.
   6. La terminal.
   7. `docs/pc2/04-analisis-speedup-escalabilidad.md`.
   8. `docs/tp/03-informe-gaps-ia.md`.
   9. `docs/tp/04-conclusiones.md`.
6. Ensayar una vez con cronómetro. Cada integrante debe aparecer con cámara o
   al menos con su voz, y presentarse por su nombre.

## Bloque 1 — Presentación, problema y dataset (0:00–1:30)

**Pantalla:** `README.md` en GitHub, secciones "Qué hacemos" y "El pipeline".

**Texto sugerido (adaptarlo a sus propias palabras):**

> Hola, somos el Grupo 5 de Programación Concurrente: [nombres]. Nuestro
> proyecto estima el costo total de una hospitalización con una regresión
> lineal multivariable, como apoyo a la planificación presupuestaria de los
> hospitales, dentro del ODS 3, Salud y bienestar.
>
> Usamos el dataset SPARCS 2022 de Nueva York, con 2 103 433 altas
> hospitalarias. Después de la limpieza conservamos 2 103 432. Codificamos las
> variables categóricas con one-hot y obtuvimos 49 características. Separamos
> 80/20 de forma reproducible y estandarizamos solo con el conjunto de
> entrenamiento para no filtrar información del de prueba.
>
> El modelo se entrena con Batch Gradient Descent: 100 épocas con learning
> rate 0.01. Como cada época recorre 1.68 millones de registros, el cálculo del
> gradiente es el cuello de botella, y por eso lo paralelizamos.

**Transición:** "Ahora [Integrante 2] explica cómo lo paralelizamos y cómo lo
verificamos."

## Bloque 2 — Worker Pool y verificación en Spin (1:30–3:30)

**Pantalla 1 (1:30–2:30):** el diagrama del README y luego `concurrent.go`:
- Líneas 113–118: canales `jobs` y `results` y el `sync.WaitGroup`.
- Línea 169: reducción por `JobID`.
- Línea 173: `wg.Wait()`.

> Usamos un Worker Pool. En cada época el conjunto se divide en `workers × 4`
> jobs que se envían por un canal. Cada worker calcula un gradiente parcial y
> lo devuelve por otro canal. Los workers solo leen los pesos y nunca los
> modifican.
>
> El coordinador espera con `wg.Wait()`, suma los gradientes ordenados por
> `JobID` y recién ahí actualiza los pesos, en un solo flujo. Por eso no
> necesitamos `Mutex`: no hay escritura compartida. Además, el orden fijo de la
> suma hace que el resultado sea idéntico al secuencial.

**Pantalla 2 (2:30–3:30):** `promela/regression_workers.pml`, en el
`proctype Worker` (línea 93) y las tres fórmulas LTL (líneas 307–309). Después,
`results/promela/w3_j4_ltl_mutex.txt` con la línea `errors: 0` y la tabla de
mutantes de `docs/tp/01-verificacion-formal.md`.

> Para demostrar que la sincronización es correcta modelamos el Worker Pool en
> Promela y lo verificamos con Spin. Verificamos tres propiedades:
> - `mutex`: nunca hay más de un proceso actualizando los pesos.
> - `safe_update`: solo se actualiza cuando llegaron todos los resultados.
> - `termination`: el entrenamiento siempre termina.
>
> La corrida de seguridad no reporta *invalid end states*, es decir, no hay
> deadlocks. Todo da `errors: 0` con 2, 3 y 4 workers.
>
> Para comprobar que la verificación no es trivial, creamos mutantes con
> errores a propósito. En uno, los workers escriben los pesos, y Spin encuentra
> la violación de exclusión mutua. En otro falta un mensaje de STOP, y Spin
> detecta el deadlock. Es decir, Spin sí detecta los errores cuando existen.

**Transición:** "Con la corrección demostrada, [Integrante 3] muestra el
rendimiento."

## Bloque 3 — Resultados y análisis con IA (3:30–5:15)

**Pantalla 1 (3:30–4:30):** la terminal con la salida de
`go run . -mode=quick` (acelerada) y después la tabla de speedup de
`docs/pc2/04-analisis-speedup-escalabilidad.md`.

> En un Ryzen 5 9600X, con 6 núcleos físicos y 12 lógicos, la versión
> secuencial tarda 1869 ms. Con 16 workers baja a 383 ms, un speedup de 4.88x.
> Las dos versiones producen exactamente el mismo modelo: diferencia máxima 0
> y R² de 0.4679 en ambas.
>
> El speedup se estanca cerca de 4.9x. La métrica de Karp–Flatt crece de 0.018
> a 0.180, lo que indica que el límite no está en una parte secuencial del
> código sino en el hardware: los hilos SMT comparten las unidades de cálculo y
> el ancho de banda de memoria. Por eso el punto de equilibrio está en
> 8 workers por eficiencia y en 12 como elección práctica.

**Pantalla 2 (4:30–5:15):** `docs/tp/02-prompt-analisis-ia.md` (el prompt) y la
tabla resumen de `docs/tp/03-informe-gaps-ia.md`.

> Para el análisis con IA escribimos un prompt estructurado con rol, alcance,
> criterios y formato de salida, y lo ejecutamos con Claude sobre el código de
> GitHub. Encontró 19 GAPs: ninguno de severidad alta, 4 de media y 15 de baja.
> Corregimos los importantes con tests: timeout de inactividad en la descarga,
> código de salida 1 ante errores, escritura atómica del CSV, validación de
> columnas y errores de cierre de archivos. Verificamos cada hallazgo contra
> el código antes de corregirlo, porque la IA también puede equivocarse.

## Bloque 4 — Discusión crítica (5:15–6:00, unos 15 s cada uno)

- **Integrante 1, limitaciones:** "Medimos en una sola máquina. Spin solo puede
  verificar configuraciones pequeñas, hasta 4 workers, por la explosión de
  estados. Y el R² de 0.47 indica que una regresión lineal explica menos de la
  mitad de la variación del costo."
- **Integrante 2, escalabilidad:** "Más workers no es mejor. Pasar de 8 a 16
  apenas mejora el tiempo y aumenta la memoria. En una máquina con más núcleos
  físicos el techo debería subir."
- **Integrante 3, mejoras:** "Proponemos mini-batch para escalar a datasets más
  grandes, reutilizar el pool de workers entre épocas e integración continua
  con `go test -race` y Spin en cada pull request. Gracias."

## Publicación

1. Exportar en MP4, 1080p, y confirmar que la duración es **≤ 6:00**.
2. Subirlo a YouTube como **no listado**, o a Google Drive con permiso
   **"Cualquier persona con el enlace"**.
3. Abrir el enlace en una ventana de incógnito para confirmar que se ve sin
   iniciar sesión.
4. Reemplazar `[ENLACE DEL VIDEO]` en el anexo de los tres Word.
