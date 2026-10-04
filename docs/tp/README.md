# TP — Evidencia por punto del enunciado

Entregable 3 (semana 7) de `Documentos/CC65_PCs_TP-202620.pdf`. Este índice
relaciona cada punto del enunciado con su evidencia en el repositorio.

| Punto | Pts | Requisito | Evidencia | Estado |
|---|---:|---|---|---|
| p | — | El informe inicia con los puntos de la PC1 y la PC2 y sus correcciones | Informe Word del TP (secciones 1 a 13 y "Correcciones respecto a la PC2") | Listo |
| q | 4 | Verificación formal en Spin: ausencia de deadlocks y exclusión mutua | [`01-verificacion-formal.md`](01-verificacion-formal.md), `promela/`, `results/promela/`, `results/promela/mutants/` | Listo |
| r | 5 | Prompt estructurado, modelo de IA y análisis del código en un informe `.md` con los GAPs | [`02-prompt-analisis-ia.md`](02-prompt-analisis-ia.md), [`03-informe-gaps-ia.md`](03-informe-gaps-ia.md) | Listo (GAPs principales corregidos) |
| s | 4 | Análisis con conclusiones y recomendaciones del alumno | [`04-conclusiones.md`](04-conclusiones.md) | Borrador: **cada integrante debe reescribirlo con su opinión** |
| t | — | Referencias bibliográficas (APA) | Informe Word del TP, sección de referencias | Listo |
| u | — | Código público en GitHub con historial de commits en la rama principal | `main` del repositorio | Listo; no editar `main` después de la fecha de entrega (−10 pts) |
| v | 5 | Video de máximo 6 minutos con la participación de todos | Guion en [`05-guion-video.md`](05-guion-video.md) | **Pendiente: grabar y pegar el enlace en el anexo del informe** (−5 pts si falta) |
| gitflow | 2 | Historial gitflow con la participación de todos los integrantes | `git shortlog -sne main` | **Pendiente: commits propios de José y Lucero** |

## Acciones pendientes del equipo

Estas tareas no se pueden automatizar porque dependen de cada integrante:

1. **Participación en GitHub:** José (`SeuNg720p`) y Lucero (`YuhiTTo`) deben
   hacer commits desde sus cuentas. Una opción útil: correr
   `go run . -mode=benchmark -runs=7 -trim=0.15` en la máquina del informe
   (Ryzen 5 9600X) y commitear `results/benchmark/`, que además agrega la
   desviación estándar al análisis.
2. **Conclusiones propias:** reescribir `04-conclusiones.md` y la sección
   equivalente del Word con la opinión de cada uno.
3. **Video:** grabarlo siguiendo el guion, subirlo y reemplazar
   `[ENLACE DEL VIDEO]` en el anexo del informe.
4. **Entrega individual:** cada integrante sube su Word
   `CC65-TP-202620-[código]` al aula virtual.
5. **Participación:** el coordinador completa y adjunta
   `Documentos/CC65-Participacion-202620.docx`.
6. **No modificar `main` después de la fecha de entrega.**
