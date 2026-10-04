# 05 — Guion del video (TP, punto v)

Requisito: video de **máximo 6 minutos** en el que **cada integrante** demuestre
conocimiento del tema y presente los resultados, con una discusión crítica
sobre limitaciones, escalabilidad y posibles mejoras. El enlace se publica en
la nube (por ejemplo YouTube no listado o Google Drive con acceso por enlace) y
va como anexo del informe. Si falta el video, se descuentan 5 puntos.

Reparto sugerido: cada integrante presenta unos 2 minutos. Ajústenlo según quién
trabajó cada parte.

| Tiempo | Integrante | Tema | Qué mostrar en pantalla |
|---|---|---|---|
| 0:00–0:30 | Integrante 1 | Presentación del grupo, el caso de uso (costo hospitalario, ODS 3) y el dataset SPARCS 2022 (2.1 M registros) | README del repositorio |
| 0:30–2:00 | Integrante 1 | Algoritmo concurrente: Worker Pool, canales, `sync.WaitGroup`, reducción por `JobID` y por qué no hace falta `Mutex` | Diagrama del README y `concurrent.go` |
| 2:00–3:30 | Integrante 2 | Verificación formal: modelo Promela, propiedades LTL `mutex`, `safe_update` y `termination`, ausencia de deadlock y mutantes detectados | `promela/regression_workers.pml`, `docs/tp/01-verificacion-formal.md`, ejecución de `run_spin_mutants.sh` |
| 3:30–5:00 | Integrante 3 | Resultados: ejecución en vivo de `go run . -mode=quick`, tabla de speedup y eficiencia, Karp–Flatt y punto de equilibrio | Terminal y tablas de `docs/pc2/` |
| 5:00–6:00 | Los tres | Discusión crítica: GAPs encontrados con IA y cómo se corrigieron, limitaciones (una sola máquina, N pequeño en Spin, R² = 0.47) y mejoras propuestas | `docs/tp/03-informe-gaps-ia.md` y `docs/tp/04-conclusiones.md` |

## Comandos para la demostración en vivo

```bash
go run . -mode=quick                 # secuencial vs. concurrente + equivalencia
go test -count=1 ./...               # suite de tests
# Spin con Docker (modelo correcto y mutantes):
docker run --rm -v "$PWD":/work -w /work debian:stable-slim bash -ec \
  'apt-get update -qq && apt-get install -y -qq spin gcc libc6-dev && \
   bash promela/run_spin.sh 3 4 && bash promela/run_spin_mutants.sh'
```

## Recomendaciones de grabación

- Que todos aparezcan con cámara o voz; el enunciado exige que **cada
  integrante** demuestre conocimiento.
- Dejar listas las terminales: el dataset ya descargado y la imagen de Docker
  ya bajada, para que la demo no espere descargas.
- Controlar el tiempo: más de 6 minutos incumple el requisito.
- Subir el video antes de la fecha de entrega y probar el enlace en una
  ventana privada.
