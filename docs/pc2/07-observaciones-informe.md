# 07 — Observaciones y mejoras sugeridas para el informe PC2

Audiencia: integrantes del equipo que editan el documento Word del
informe (`Documentos/CC65-PC2-202620-Grupo5.pdf` es la exportación
actual a PDF). Este documento no modifica el informe; solo reúne
observaciones y **textos listos para pegar** en las secciones
correspondientes.

## 1. Checklist de rúbrica vs. informe

| Punto | Descripción | Sección del informe | Estado |
|---|---|---|---|
| (a) | Modelo Promela, libre de condiciones de carrera | §6 | listo en el informe; además, las propiedades LTL `safe_update`, `termination` y `mutex` se verificaron con Spin 6.5.2 en las variantes 2/4, 3/4 y 4/4, todas con `errors: 0` (tabla en `02-modelo-promela.md`). Opcional: agregar esa tabla al final de §6.4 como evidencia de exclusión mutua, lo que adelanta trabajo del TP |
| (b) | Go secuencial vs. concurrente con evidencia | §7 (Fig. 10) | listo |
| (c) | Sincronización (Worker Pools/Pipelines) | §7.4–§7.5 | listo, pero ver corrección de PC1 sobre `sync.Mutex` en el bloque 2(a) abajo |
| (d) | Speedup, media recortada, tabla + estadística | §8–§9 | tabla y speedup/eficiencia listos; falta desviación estándar/CV (ver bloque 2(b)) |
| (e) | Análisis de speedup/escalabilidad/trade-offs | §10 | falta Karp–Flatt y contraste con hardware (ver bloque 2(b)) |
| (f) | Recursos hasta el punto de equilibrio | §11 | listo, pero el criterio de elección de 12 workers puede reforzarse (ver bloque 2(c)) |
| (g) | Historial de gitflow | §12 | falta detalle por integrante (ver bloque 2(d)) |

## 2. Textos listos para pegar

### (a) Nueva sección "Correcciones respecto a la PC1"

> **Correcciones respecto a la PC1**
>
> Esta versión del informe corrige los siguientes puntos identificados
> tras la retroalimentación de la PC1:
>
> - El resumen y los objetivos, escritos originalmente en tiempo
>   futuro como planificación del trabajo, se reescribieron en tiempo
>   pasado para describir el trabajo efectivamente realizado y sus
>   resultados.
> - La §3.2 y las opiniones críticas asociadas ya no afirman que la
>   sincronización usa `sync.Mutex`. La implementación final usa
>   canales (`jobs`/`results`) más `sync.WaitGroup` para coordinar a
>   los workers, sin `Mutex`: cada worker calcula su gradiente parcial
>   de forma aislada y lo envía por canal; la reducción y la
>   actualización de pesos ocurren de forma secuencial y determinista
>   en el coordinador después del `wg.Wait()`, no por combinación
>   concurrente de sumas parciales protegida por un mutex (que era lo
>   que planteaba la PC1 como diseño previsto).
> - El párrafo sobre la cadena de búsqueda de la §3.3, que aparecía
>   duplicado en la PC1, se consolidó en una sola versión.
> - Todo el documento se revisó para usar tiempo pasado de forma
>   consistente, salvo las secciones que explícitamente describen
>   trabajo futuro o limitaciones.
> - La opinión crítica sobre el Paper 2 se reescribió para vincularla
>   con el diseño real implementado (worker pool con reducción
>   determinista por `JobID`) en lugar de la referencia genérica de la
>   PC1.
> - La opinión sobre el Paper 3 se amplió para señalar que, en esta
>   implementación, los workers nunca escriben directamente sobre los
>   pesos globales del modelo: solo devuelven gradientes parciales: la
>   actualización de pesos es responsabilidad exclusiva del
>   coordinador.

### (b) Párrafo para §10 (Karp–Flatt e interpretación)

> **Estimación de la fracción secuencial (Karp–Flatt)**
>
> A diferencia de la ley de Amdahl, que requiere conocer de antemano
> la fracción secuencial `f`, la métrica de Karp–Flatt permite
> estimarla experimentalmente a partir del speedup medido:
> `e = (1/S(p) - 1/p) / (1 - 1/p)`.
>
> | p | Speedup(p) | e (Karp–Flatt) |
> |---|---|---|
> | 2 | 1.9655 | 0.0176 |
> | 4 | 3.6615 | 0.0308 |
> | 8 | 4.7050 | 0.1000 |
> | 12 | 4.8224 | 0.1353 |
> | 16 | 4.8817 | 0.1518 |
> | 24 | 4.8603 | 0.1712 |
> | 32 | 4.8668 | 0.1799 |
>
> `e` crece de forma monótona con `p` en lugar de mantenerse estable,
> lo que indica que la meseta de speedup (~4.8x–4.9x) no se explica
> por una fracción secuencial fija del algoritmo (Amdahl con
> `f ≈ 0.02–0.03` permitiría speedups teóricos de 30x o más). La causa
> real es el límite de hardware: la máquina de prueba tiene 6 núcleos
> físicos (12 lógicos vía SMT); la curva de speedup se dobla justo
> después de los 6 núcleos físicos (S(4)=3.66 → S(8)=4.70, y luego se
> aplana: S(12)=4.82, S(16)=4.88, S(24)=4.86, S(32)=4.87), porque los
> hilos lógicos adicionales comparten unidades de ejecución de punto
> flotante/SIMD y ancho de banda de memoria con su núcleo físico —
> coherente con que el cálculo de gradientes sobre ~1.68M filas es una
> carga limitada por ancho de banda de memoria, no por cómputo puro.

### (c) Párrafo complementario para §11.3 (punto de equilibrio)

> **Complemento al criterio de punto de equilibrio**
>
> Aplicando las reglas automáticas del propio proyecto
> (`findEquilibrium` al 95 % del speedup máximo y `findEfficiencyDrop`
> al 50 % de eficiencia, `report.go`), el punto de equilibrio
> eficiencia-óptimo es `workers=8`: alcanza 4.7050x (96.4 % del
> speedup máximo de 4.8817x en 16 workers) con eficiencia 0.5881 (la
> última configuración con `Efficiency ≥ 0.5`) y un 31 % menos
> asignaciones que 12 workers (4,610 vs. 6,625 mallocs). La elección
> de `workers=12` de este informe es una decisión válida orientada a
> throughput —coincide con el número de CPUs lógicas de la máquina y
> compra un +2.5 % adicional de speedup (4.7050→4.8224) a cambio de
> más memoria/asignaciones—, mientras que 8 workers representa el
> mejor compromiso costo-beneficio según el criterio cuantitativo de
> eficiencia. Se recomienda declarar explícitamente ambos criterios en
> el informe final: `workers=8` como equilibrio eficiencia-óptima y
> `workers=12` como elección orientada a throughput/paralelismo
> disponible.

### (d) Nota para §12 (Evidencia de commits / GitHub)

> **Evidencia de commits por integrante**
>
> Ramas y pull requests relevantes de este ciclo:
> - `feature/pc2-evidence` → PR #4 (mergeado a `develop`).
> - `feature/dataset-download` → PR #5 (mergeado a `develop`).
> - `develop` → `main` → PR #6.
>
> El detalle de commits debe mostrarse por cuenta de GitHub, no solo
> por nombre, ya que las cuentas del equipo son:
> `Jaed69` = Jhamil, `SeuNg720p` = José, `YuhiTTo` = Lucero.
>
> Para capturar el resumen de commits por autor:
> ```bash
> git shortlog -sne --all
> ```
> Incluir la captura de esta salida (o de la vista "Contributors" de
> GitHub) como evidencia de §12, en vez de solo el `git log --graph`.

## 3. Riesgos para la entrega

| Riesgo | Detalle | Acción sugerida |
|---|---|---|
| Formato de entrega individual | La rúbrica pide un Word por alumno nombrado `CC65-PC2-202620-[código de alumno]`, subido individualmente; el repositorio solo tiene un PDF único `...-Grupo5.pdf` | El coordinador debe generar/subir una copia por integrante con el nombre de archivo correcto antes de la fecha límite |
| Anexo de participación | Debe adjuntarse `CC65-Participación-202620` | Verificar que el coordinador lo adjunte junto con el informe |
| PDF de la PC1 removido | El PDF de la PC1 se eliminó de `Documentos/` en el commit `a630ff3` | Se sugiere conservarlo (o restaurarlo) para trazabilidad de las correcciones del bloque 2(a); no es indispensable para la entrega de PC2 |
| "3 papers por integrante" vs. informe | El texto del Entregable 1 pide "3 papers top ranking por cada integrante"; el informe documenta 1 paper por integrante | Esto es una posible discrepancia de alcance, no un hecho de calificación confirmado — se recomienda confirmar con el profesor antes de asumir que falta contenido |
| Captura de commits en §12 | La sección "Evidencia de commits" debe contener la captura de pantalla real (ver bloque 2(d)), no solo el comando | Tomar la captura antes de la entrega final |
| Ecuaciones (§7–§8) | Las fórmulas de estas secciones no se pudieron extraer como texto del PDF (típico de los objetos de ecuación de Word, no necesariamente un error) | Revisar visualmente en el Word que se vean bien antes de entregar |
