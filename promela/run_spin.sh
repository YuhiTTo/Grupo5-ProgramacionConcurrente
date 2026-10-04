#!/usr/bin/env bash
#
# Ejecuta la verificación formal (Spin + pan) del modelo
# promela/regression_workers.pml: primero seguridad
# (deadlock / invalid end states / assertion violations)
# y luego cada propiedad LTL por separado (-N <nombre>).
#
# Requiere `spin` y `gcc` instalados y en el PATH. Si no están
# disponibles localmente, ejecutarlo en Docker con el comando
# documentado en el README (sección de verificación formal) y al
# final de este archivo.
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

for arg_name in WORKERS JOBS; do
    if ! [[ "${!arg_name}" =~ ^[1-9][0-9]*$ ]]; then
        echo "Error: ${arg_name} debe ser un entero positivo (recibido: '${!arg_name}')" >&2
        echo "Uso: $0 [NUM_WORKERS] [NUM_JOBS]" >&2
        exit 2
    fi
done

for tool in spin gcc; do
    if ! command -v "$tool" > /dev/null 2>&1; then
        echo "Error: '$tool' no está instalado o no está en el PATH." >&2
        echo "Instálelo (apt-get install -y spin gcc libc6-dev) o use el comando Docker del README." >&2
        exit 127
    fi
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODEL="$SCRIPT_DIR/regression_workers.pml"
OUT_DIR="$SCRIPT_DIR/../results/promela"
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

mkdir -p "$OUT_DIR"

PREFIX="w${WORKERS}_j${JOBS}"

# pan termina con código 0 aunque encuentre violaciones; el
# resultado real está en la línea "errors: N" de su salida.
require_no_errors() {
    if ! grep -qE "errors: 0([^0-9]|$)" "$1"; then
        echo "FALLO: $1 no reporta 'errors: 0'" >&2
        exit 1
    fi
}

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
require_no_errors "$OUT_DIR/${PREFIX}_safety.txt"

# 3) Compilar (sin -DSAFETY) y correr cada propiedad LTL
#    por separado con -N <nombre>. -a mantiene la búsqueda
#    de invalid end states también durante la verificación
#    de la fórmula LTL.
gcc -o pan pan.c

for ltl_name in safe_update termination mutex; do
    echo "-- LTL: ${ltl_name} --"
    ./pan -a -N "${ltl_name}" | tee "$OUT_DIR/${PREFIX}_ltl_${ltl_name}.txt"
    require_no_errors "$OUT_DIR/${PREFIX}_ltl_${ltl_name}.txt"
done

popd > /dev/null

echo "Listo. Resultados en $OUT_DIR"

#
# Alternativa con Docker (si spin/gcc no están disponibles
# localmente); es el comando verificado por el equipo:
#
#   docker run --rm -v "$PWD":/work -w /work debian:stable-slim bash -ec \
#     'apt-get update -qq >/dev/null && \
#      apt-get install -y -qq spin gcc libc6-dev >/dev/null 2>&1 && \
#      bash promela/run_spin.sh 2 4'
#
