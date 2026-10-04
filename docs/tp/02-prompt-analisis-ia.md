# Prompt estructurado para el análisis de código con IA

Entregable del TP, punto (r): redactar un prompt estructurado, seleccionar un
modelo de IA, analizar el código del repositorio en GitHub y producir un
informe `.md` con el detalle técnico de los GAPs existentes en calidad de
código, seguridad, patrones de concurrencia u otros aspectos relevantes.

El informe resultante se encuentra en
[`03-informe-gaps-ia.md`](03-informe-gaps-ia.md).

## 1. Modelo seleccionado

| Campo | Valor |
|---|---|
| Modelo | Claude (familia Claude 5) |
| Herramienta de ejecución | Claude Code (agente en línea de comandos con acceso al repositorio) |
| Fecha del análisis | 2026-10-03 |
| Commit analizado | `cd142a2` (obtenido con `git rev-parse --short HEAD`, rama `feature/tp-delivery`) |

### Justificación de la selección

1. **Contexto largo.** El repositorio contiene unas 7 000 líneas entre código
   Go, pruebas y modelos Promela. Un contexto amplio permite leer cada archivo
   completo en lugar de analizar fragmentos aislados, lo que es necesario para
   razonar sobre interacciones entre archivos (por ejemplo, `main.go`,
   `concurrent.go` y `benchmark.go`).
2. **Uso de herramientas.** Claude Code puede leer archivos con numeración de
   líneas (requisito para citar `archivo:línea`) y ejecutar `go vet`,
   `go test` y `gofmt`, de modo que los hallazgos se contrastan con evidencia
   ejecutable y no solo con lectura estática.
3. **Razonamiento sobre código concurrente.** El análisis exige seguir el
   flujo de canales, `sync.WaitGroup` y reducción determinista, y verificar
   invariantes (capacidad de buffers, orden de escritura de pesos), tarea en
   la que importa más el razonamiento paso a paso que la coincidencia de
   patrones.

## 2. Prompt estructurado

El texto siguiente es el prompt utilizado, sin modificaciones.

````text
# ROL
Eres un ingeniero de software senior especializado en Go, programación
concurrente y revisión de código (code review y auditoría técnica). Tu
objetivo es identificar GAPs reales y verificables, no inflar la lista.

# CONTEXTO
Proyecto universitario del curso de Programación Concurrente (CC65).
Implementa regresión lineal multivariada con descenso de gradiente por lotes
sobre el dataset SPARCS 2022 (aprox. 2.1 millones de filas, 49 features),
en Go (solo biblioteca estándar, sin dependencias externas). Existen dos
variantes de entrenamiento:
- Secuencial (sequential.go).
- Concurrente con worker pool (concurrent.go): canales jobs/results,
  sync.WaitGroup y reducción determinista por JobID.
Además hay un CLI con modos (quick, benchmark, resources, cpu-profile, clean,
download, all), descarga verificada del dataset, estadísticas de benchmark,
reportes en CSV/Markdown y modelos Promela verificados con Spin.

# ALCANCE
Analiza TODO el repositorio:
- Código Go no de prueba: main.go, config.go, dataset.go, cleaning.go,
  dataset_download.go, preprocessing.go, scaling.go, regression.go,
  sequential.go, concurrent.go, benchmark.go, stats.go, report.go,
  resource_profile.go, cpu_profile.go, equivalence.go.
- Pruebas: *_test.go (evalúa cobertura y calidad, a nivel general).
- Verificación formal: promela/*.pml, promela/run_spin.sh,
  promela/run_spin.ps1, promela/run_spin_mutants.sh.
Lee cada archivo completo antes de emitir juicios.

# CRITERIOS DE ANÁLISIS
1. Calidad y mantenibilidad: duplicación, código muerto o inalcanzable,
   valores fijos en el código, claridad de nombres, formato (gofmt).
2. Seguridad: validación de entradas, descarga de red (HTTPS, allowlist,
   redirecciones, timeouts, verificación de integridad), manejo de rutas y
   permisos de archivos, uso de scripts shell.
3. Patrones de concurrencia y sincronización: data races, deadlocks, fugas
   de goroutines, disciplina de canales (quién cierra, capacidad de buffers),
   uso de WaitGroup, orden de escritura de estado compartido, determinismo.
4. Rendimiento y memoria: asignaciones por época, creación de goroutines,
   pasadas redundantes sobre los datos, sobrecarga de la instrumentación.
5. Manejo de errores y robustez: errores ignorados, envoltura de errores
   (%w), códigos de salida, limpieza de recursos ante fallos, validación de
   parámetros de la CLI (workers, trim, runs, epochs), divisiones por cero.
6. Pruebas y verificación: archivos sin pruebas, casos borde no cubiertos,
   alcance de la verificación formal frente al código real.
7. Reproducibilidad: semillas, metadatos del entorno, dependencia de rutas o
   versiones, scripts de verificación repetibles.

# DEFINICIÓN DE SEVERIDAD
- Alta: puede producir resultados incorrectos, pérdida o corrupción de
  datos, vulnerabilidad explotable, deadlock o data race comprobable.
- Media: degrada la robustez, la seguridad o la mantenibilidad de forma
  apreciable, pero con mitigación parcial o impacto acotado al uso normal.
- Baja: mejora de estilo, claridad o eficiencia sin impacto funcional
  observable; deuda técnica menor.

# FORMATO DE SALIDA
Para cada GAP entrega una fila con estos campos:
- ID (GAP-01, GAP-02, ...)
- Categoría (uno de los criterios anteriores)
- Severidad (Alta / Media / Baja)
- Evidencia (archivo:línea, verificada leyendo el código)
- Descripción
- Impacto
- Recomendación
Incluye además: resumen ejecutivo con conteos por severidad y categoría,
sección de fortalezas (aspectos correctos que NO son GAPs), priorización y
limitaciones del análisis.

# RESTRICCIONES
- Basa cada afirmación en evidencia: cita siempre archivo:línea.
- Distingue explícitamente HECHOS VERIFICADOS (leídos en el código o
  confirmados con una ejecución) de HIPÓTESIS (no comprobadas).
- No inventes problemas ni líneas. Si una sospecha se refuta al leer el
  código, repórtala como refutada o inclúyela en Fortalezas.
- No infles la severidad. Un GAP sin impacto demostrable es Baja.
- No modifiques el código analizado; este es un análisis de solo lectura.
- Ejecuta go vet y go test (y go test -race si el entorno lo permite) y
  reporta los resultados reales.
- Entrega el informe en español profesional y neutro, en formato Markdown.

# HIPÓTESIS A CONFIRMAR O REFUTAR CON EVIDENCIA
a) Falta de timeout de inactividad/lectura al transmitir el cuerpo de la
   descarga (dataset_download.go).
b) Errores ignorados de Flush/Error de csv.Writer o de Write/Close de
   archivos (report.go).
c) Limpieza de recursos en cpu_profile.go si algo falla a mitad de la
   ejecución.
d) Claridad y corrección del dimensionamiento de buffers de los canales
   jobs/results en concurrent.go.
e) Robustez de promela/run_spin.ps1 y promela/run_spin.sh.
````

## 3. Procedimiento de ejecución

1. Se fijó el commit analizado con `git rev-parse --short HEAD`, que devolvió
   `cd142a2`, sobre la rama `feature/tp-delivery`.
2. Se entregó el prompt anterior al modelo mediante Claude Code, con el
   repositorio como directorio de trabajo.
3. El modelo leyó completos todos los archivos Go no de prueba, los scripts de
   Spin y, a nivel de inventario, los archivos de prueba (nombres de pruebas y
   funciones cubiertas).
4. Se ejecutaron `go vet ./...`, `gofmt -l` y `go test -count=1 ./...` en el
   entorno local. La ejecución con detector de carreras
   (`go test -race -count=1 ./...`) se hizo previamente en Docker
   (`golang:1.27`), porque requiere CGO.
5. Cada hipótesis (a) a (e) se contrastó con el código y se clasificó como
   confirmada, parcialmente confirmada o refutada; el resultado se documenta en
   el informe.
6. Se revisaron manualmente las citas `archivo:línea` antes de incluirlas en
   el informe.

## 4. Salida

El producto de este procedimiento es
[`03-informe-gaps-ia.md`](03-informe-gaps-ia.md), que contiene el resumen
ejecutivo, la metodología, las fortalezas, la tabla de GAPs, el detalle por
GAP, la priorización y las limitaciones.
