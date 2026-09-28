# 06 — Checklist de evidencias (capturas de pantalla)

Guardar las capturas en `docs/pc2/evidencias/` con nombres
descriptivos (p. ej. `01-quick-comparacion.png`). Este documento lista
qué capturar y con qué comando generarlo.

## (b) Go secuencial vs. concurrente

- [ ] **Comparación rápida** (secuencial vs. concurrente, resultados
      de Test + speedup preliminar + verificación de equivalencia):
      ```bash
      go run . -mode=quick -epochs 100
      ```
      Capturar la sección " COMPARACIÓN INICIAL" completa (tiempos,
      speedup, diferencia máxima entre modelos, verificación de
      equivalencia OK).

## (d) Benchmark formal / speedup / estadísticas

- [ ] **Corrida del benchmark formal completo**:
      ```bash
      go run . -mode=benchmark -runs 10 -epochs 100 -warmup 1 \
        -workers 1,2,4,8,12,16,24,32
      ```
      Capturar la salida de consola (" RESULTADOS DEL BENCHMARK") y
      las líneas "guardado en: ..." que confirman la persistencia.
- [ ] **Contenido de `results/benchmark/speedup_summary.md`** (tabla
      completa, puede ser captura del archivo renderizado o del CSV
      abierto en una hoja de cálculo).
- [ ] **Contenido de `results/benchmark/equilibrium.md`**.

## (f) Recursos y punto de equilibrio

- [ ] **Corrida del perfil de recursos**:
      ```bash
      go run . -mode=resources -epochs 100
      ```
      Capturar la salida " RESUMEN DE RECURSOS".
- [ ] **`results/resources/resources.csv`** (tabla, captura o
      screenshot de hoja de cálculo).
- [ ] **Perfil de CPU**:
      ```bash
      go run . -mode=cpu-profile -epochs 100
      go tool pprof -top results/cpu/cpu.prof
      ```
      Capturar el top de funciones por tiempo de CPU.

## (a) / (g) Promela y gitflow

- [ ] **Corridas de Spin** (una captura por variante/propiedad, o el
      contenido de los `.txt` en `results/promela/`):
      ```bash
      ./promela/run_spin.sh        # o run_spin.ps1 en Windows
      ```
      Capturar al menos: `errors: 0` de la verificación de seguridad,
      y el resultado de cada propiedad LTL (`safe_update`,
      `termination`, `mutex`).
- [ ] **Historial de gitflow**:
      ```bash
      git log --graph --oneline --all
      ```
      Capturar el árbol de ramas/commits que muestre el flujo
      feature → develop/main.

## Notas

- Todas las corridas de (b), (d) y (f) requieren el dataset real
  (`dataset/SPARCS_2022_clean_go.csv`, ver README raíz) — no están
  incluidas en este repositorio por tamaño.
- Las corridas de Promela requieren Spin + gcc instalados (ver
  `02-modelo-promela.md`).
