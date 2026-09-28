# Dataset: Hospital Inpatient Discharges (SPARCS)

Este directorio está destinado a almacenar el dataset de altas hospitalarias del estado de Nueva York (**SPARCS** - *Statewide Planning and Research Cooperative System*), publicado por el New York State Department of Health a través de Health Data NY.

Debido a restricciones de tamaño de GitHub (límite de 100 MB por archivo; el archivo CSV original supera los 900 MB), los archivos `.csv` en esta carpeta están excluidos del control de versiones mediante `.gitignore`. Cada quien que clone el repositorio necesita obtener los archivos por su cuenta; este README documenta la forma automática (recomendada) y la manual.

## Descarga automática (recomendada)

El binario del proyecto sabe descargar y verificar los datasets por sí mismo, sin depender de nada más que la librería estándar de Go:

```bash
go run . -mode=download                 # dataset limpio (default), a dataset/SPARCS_2022_clean_go.csv
go run . -mode=download -download=raw   # dataset original (crudo), a la ruta de cleaning.go
go run . -mode=download -download=all   # ambos
```

Además, cualquier modo que necesite el dataset limpio (`quick`, `benchmark`, `resources`, `cpu-profile`, `all`) lo descarga automáticamente si falta y `-dataset` sigue apuntando a la ruta por defecto; `-mode=clean` hace lo mismo para el dataset original. Para desactivar la descarga automática y obtener en su lugar un error accionable, use `-no-download`.

### Qué se verifica antes de aceptar un archivo

- **HTTPS únicamente** y el host final (tras redirecciones) debe estar en una lista de hosts permitidos (Google Drive y su CDN de contenido); cualquier otro esquema u host se rechaza.
- Se rechaza una respuesta que no sea `200 OK` o cuyo `Content-Type` sea `text/html` (la página de aviso/cuota que Drive muestra cuando no hay confirmación automática).
- El tamaño anunciado por el servidor (`Content-Length`) y el tamaño final descargado deben coincidir exactamente con el tamaño esperado.
- El **SHA-256** del contenido descargado debe coincidir exactamente con el valor fijado (ver `SHA256SUMS` en este directorio).
- La escritura es atómica: se descarga a `<archivo>.part` y solo se renombra al nombre final si todas las verificaciones anteriores pasan. Ante cualquier fallo, el `.part` se elimina — nunca queda un dataset corrupto en su ruta final.
- Si el archivo ya existe y pasa la verificación de tamaño + hash, no se vuelve a descargar.

### Alternativa manual

1. Acceder a los enlaces de Google Drive en [`Link del Dataset Original y Limpio.txt`](./Link%20del%20Dataset%20Original%20y%20Limpio.txt) (también listados en el README raíz del proyecto).
2. Descargar el archivo `.csv` correspondiente.
3. Colocarlo en esta carpeta `dataset/`, con el nombre exacto que espera el código:
   - Dataset original: `Hospital_Inpatient_Discharges_(SPARCS_De-Identified)__2022_20260913.csv`
   - Dataset limpio: `SPARCS_2022_clean_go.csv`
4. Verificar manualmente su integridad contra `SHA256SUMS`:

   **Linux / macOS**

   ```bash
   cd dataset
   sha256sum -c SHA256SUMS          # Linux (coreutils)
   shasum -a 256 -c SHA256SUMS      # macOS
   ```

   **Windows (PowerShell)**

   ```powershell
   cd dataset
   Get-FileHash -Algorithm SHA256 .\SPARCS_2022_clean_go.csv
   Get-FileHash -Algorithm SHA256 ".\Hospital_Inpatient_Discharges_(SPARCS_De-Identified)__2022_20260913.csv"
   # Comparar el campo Hash contra el valor correspondiente en SHA256SUMS
   ```

### Los archivos `.csv` nunca se commitean

`dataset/*.csv` está en `.gitignore` porque GitHub bloquea archivos de más de 100 MB y ambos datasets lo superan ampliamente (el original supera los 900 MB). `SHA256SUMS` sí se versiona: es un archivo de texto pequeño que fija la integridad esperada de ambos datasets, sin necesidad de commitear los datos en sí.

### Si el archivo de Google Drive cambia alguna vez

Si el dataset publicado en Drive se reemplaza (nueva versión, corrección de datos, etc.), el hash y tamaño fijados en este proyecto quedarán obsoletos y toda descarga fallará la verificación por diseño (eso es intencional: preferimos fallar a aceptar un archivo no verificado). Para actualizar el pin:

1. Descargar el nuevo archivo manualmente y calcular su tamaño y SHA-256 (`sha256sum`, `shasum -a 256` o `Get-FileHash`).
2. Actualizar las constantes `*Size` y `*SHA256` en `dataset_download.go` (y, si cambió el Drive ID, también `*DriveID`).
3. Actualizar `dataset/SHA256SUMS` con el nuevo hash.
4. Verificar con `go test ./...` y una descarga real (`go run . -mode=download`).
