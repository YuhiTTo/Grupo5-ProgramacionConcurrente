#!/usr/bin/env bash
#
# Ejecuta la verificación formal (Spin + pan) del modelo
# promela/regression_workers.pml: primero seguridad
# (deadlock / invalid end states / assertion violations)
# y luego cada propiedad LTL por separado (-N <nombre>).
#
# Requiere `spin` y `gcc` instalados y en el PATH. NO están
# instalados en el entorno donde se generó este script;
# el equipo debe ejecutarlo localmente o vía Docker (ver
# nota al final) para producir la evidencia real.
#
# Uso:
#   ./run_spin.sh                              # NUM_WORKERS=3, NUM_JOBS=4 (defaults del modelo)
#   ./run_spin.sh 2 4                          # NUM_WORKERS=2, NUM_JOBS=4
#   ./run_spin.sh 4 4                          # NUM_WORKERS=4, NUM_JOBS=4
#   (4 8 supera 34M estados / 3 GB sin terminar; ver docs/pc2/02)
#
# Salidas guardadas en results/promela/, con nombres que
# incluyen la variante NUM_WORKERS/NUM_JOBS, p. ej.:
#   results/promela/w2_j4_safety.txt
#   results/promela/w2_j4_ltl_safe_update.txt
#   results/promela/w2_j4_ltl_termination.txt
#   results/promela/w2_j4_ltl_mutex.txt

set -euo pipefail

WORKERS="${1:-3}"
JOBS="${2:-4}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODEL="$SCRIPT_DIR/regression_workers.pml"
OUT_DIR="$SCRIPT_DIR/../results/promela"
WORK_DIR="$(mktemp -d)"

mkdir -p "$OUT_DIR"

PREFIX="w${WORKERS}_j${JOBS}"

echo "== Verificando NUM_WORKERS=${WORKERS} NUM_JOBS=${JOBS} =="

pushd "$WORK_DIR" > /dev/null

# 1) Generar el verificador C a partir del modelo, con las
#    defines de la variante actual.
spin -a -DNUM_WORKERS="${WORKERS}" -DNUM_JOBS="${JOBS}" "$MODEL"

# 2) Compilar y correr la verificación de seguridad
#    (deadlock, invalid end states, assertion violations).
#    -DNOCLAIM ignora las fórmulas ltl del modelo para que
#    esta corrida sea un chequeo puro de seguridad.
gcc -DSAFETY -DNOCLAIM -o pan pan.c
./pan | tee "$OUT_DIR/${PREFIX}_safety.txt"

# 3) Compilar (sin -DSAFETY) y correr cada propiedad LTL
#    por separado con -N <nombre>. -a mantiene la búsqueda
#    de invalid end states también durante la verificación
#    de la fórmula LTL.
gcc -o pan pan.c

for ltl_name in safe_update termination mutex; do
    echo "-- LTL: ${ltl_name} --"
    ./pan -a -N "${ltl_name}" | tee "$OUT_DIR/${PREFIX}_ltl_${ltl_name}.txt"
done

popd > /dev/null
rm -rf "$WORK_DIR"

echo "Listo. Resultados en $OUT_DIR"

#
# Alternativa con Docker (si spin/gcc no están disponibles
# localmente, p. ej. en este entorno de desarrollo):
#
#   docker run --rm -v "$PWD":/work -w /work \
#     -e WORKERS=3 -e JOBS=4 \
#     <imagen-con-spin-y-gcc> \
#     bash -c 'promela/run_spin.sh "$WORKERS" "$JOBS"'
#
# No existe todavía una imagen oficial del equipo; se puede
# construir una mínima con `apt-get install -y spin gcc` (Debian/
# Ubuntu) o usar una imagen pública con Spin preinstalado.
#
