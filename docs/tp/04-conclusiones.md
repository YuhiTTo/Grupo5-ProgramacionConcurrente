# 04 — Conclusiones y recomendaciones (TP, punto s)

> **Importante para el equipo:** el punto s pide el análisis **del alumno**.
> Este documento es un borrador basado en la evidencia del repositorio. Cada
> integrante debe revisarlo y reescribir las conclusiones y recomendaciones
> con su propia opinión antes de pasarlas al informe Word.

## Conclusiones

1. **La concurrencia mejoró el tiempo sin cambiar el resultado.** Con 16
   workers el entrenamiento bajó de 1869.45 ms a 382.95 ms (speedup 4.88x en
   un Ryzen 5 9600X) y ambas versiones produjeron exactamente el mismo modelo:
   diferencia máxima entre parámetros 0 y R² = 0.4679 en las dos. La clave fue
   la reducción determinista por `JobID`: los gradientes parciales se suman
   siempre en el mismo orden.

2. **El diseño evita la contención en lugar de administrarla.** Los workers
   solo leen los pesos y devuelven gradientes por un canal; la escritura
   ocurre en un solo flujo después de `wg.Wait()`. Por eso no hizo falta
   `sync.Mutex`, en línea con lo que muestran Schüle et al. (2022) y Bäckström
   et al. (2021): la sincronización inadecuada puede anular el beneficio del
   paralelismo.

3. **El límite de escalabilidad es el hardware, no el algoritmo.** La métrica
   de Karp–Flatt crece de 0.018 (2 workers) a 0.180 (32 workers). Si el límite
   fuera una parte secuencial fija, se mantendría constante. El speedup se
   estanca al superar los 6 núcleos físicos, porque los hilos SMT comparten
   las unidades de punto flotante y el ancho de banda de memoria.

4. **Más workers no es mejor.** 8 workers alcanzan el 96 % del speedup máximo
   con eficiencia 0.59. Pasar de 8 a 12 workers agrega un 2.5 % de speedup a
   cambio de un 44 % más de asignaciones de memoria, y pasar de 12 a 16
   agrega un 1.2 % a cambio de un 30 % más. La sobresuscripción (24 y 32
   workers) no aporta mejoras.

5. **La verificación formal tiene valor real.** Spin no encontró deadlocks ni
   violaciones de exclusión mutua en el modelo correcto, y sí detectó los tres
   mutantes con defectos introducidos a propósito. El detector de carreras de
   Go (`go test -race`) tampoco encontró carreras en el Worker Pool.

6. **El análisis con IA aportó una revisión sistemática.** Encontró 19 GAPs,
   ninguno de severidad alta. Los de severidad media se referían a robustez
   (descargas que podían colgarse, errores que terminaban con código 0,
   escrituras no atómicas) y a cobertura de pruebas, no a la lógica
   concurrente. La IA fue útil para recorrer todo el código con un criterio
   uniforme, pero cada hallazgo se verificó contra el código antes de
   corregirlo.

## Recomendaciones

1. **Mini-batch o SGD con promediado** para reducir el costo por época y
   escalar a datasets mayores, manteniendo la reducción determinista.
2. **Pool de workers persistente** entre épocas, reutilizando canales y
   buffers, para reducir las asignaciones por época (GAP-06).
3. **Medir en máquinas con más núcleos físicos** para separar el efecto de
   SMT del overhead de coordinación.
4. **Integración continua** que ejecute `go vet`, `go test -race`, los
   scripts de Spin y los mutantes en cada pull request.
5. **Estadística del benchmark en el repositorio:** correr
   `go run . -mode=benchmark -runs=7 -trim=0.15` en la máquina de referencia
   y versionar `results/benchmark/` para tener desviación estándar y
   coeficiente de variación.
6. **Mejorar el modelo, no solo la velocidad:** R² = 0.47 indica que la
   regresión lineal explica menos de la mitad de la varianza del costo;
   variables de interacción o modelos no lineales podrían mejorarlo.

## Limitaciones

- Los tiempos oficiales se midieron en una sola máquina.
- La verificación en Spin es exhaustiva solo para configuraciones pequeñas
  (hasta 4 workers y 4 jobs) por la explosión del espacio de estados.
- El análisis con IA es estático; complementa, pero no reemplaza, las
  pruebas y la verificación formal.
