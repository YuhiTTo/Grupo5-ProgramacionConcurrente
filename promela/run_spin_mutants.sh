#!/usr/bin/env bash
#
# Verifica que Spin RECHAZA los mutantes del modelo
# promela/regression_workers.pml. Un mutante es una copia del
# modelo con un unico defecto introducido a proposito; si Spin
# no lo detecta, la verificacion del modelo correcto no seria
# significativa.
#
# Mutantes:
#   race      regression_workers_race.pml
#             Los Workers tambien entran a la seccion critica de
#             actualizacion de pesos. Se espera violacion de las
#             propiedades LTL mutex y safe_update.
#   deadlock  regression_workers_deadlock.pml
#             El Coordinator envia un STOP menos que Workers. Se
#             espera "invalid end state" en la corrida de seguridad.
#
# Requiere `spin` y `gcc` en el PATH (ver alternativa con Docker
# al final y en docs/tp/01-verificacion-formal.md).
#
# Uso:
#   ./run_spin_mutants.sh    # NUM_WORKERS=2, NUM_JOBS=4
#
# Salidas en results/promela/mutants/:
#   <mutante>_<chequeo>.txt         salida de pan
#   <mutante>_<chequeo>_trail.txt   reproduccion del contraejemplo
#                                   (spin -t -p, truncada a 200 lineas)
#
# Codigo de salida: 0 solo si TODOS los mutantes son detectados
# (errors >= 1); distinto de 0 si alguno reporta "errors: 0".

set -euo pipefail

WORKERS=2
JOBS=4

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT_DIR="$SCRIPT_DIR/../results/promela/mutants"
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

mkdir -p "$OUT_DIR"

UNDETECTED=0

# pan termina con codigo 0 aunque encuentre violaciones; el
# resultado real esta en la linea "errors: N" de su salida.
# Para un mutante, "errors: 0" es el FALLO.
expect_errors() {
    local file="$1"
    if grep -qE "errors: 0([^0-9]|$)" "$file"; then
        echo "FALLO: el mutante NO fue detectado ($file reporta 'errors: 0')" >&2
        UNDETECTED=1
    elif ! grep -qE "errors: [1-9]" "$file"; then
        echo "FALLO: no se pudo leer 'errors: N' en $file" >&2
        UNDETECTED=1
    else
        echo "OK: mutante detectado ($(grep -oE 'errors: [0-9]+' "$file" | head -n1))"
    fi
}

# Replica el contraejemplo con spin -t -p (debe ejecutarse en el
# directorio donde pan dejo el .trail). $1=modelo $2=salida
# (el .trail ya registra la formula LTL usada; no hace falta -N)
replay_trail() {
    local model="$1" out="$2"
    spin -t -p -DNUM_WORKERS="${WORKERS}" -DNUM_JOBS="${JOBS}" "$model" 2>&1 \
        | head -n 200 > "$out" || true
}

pushd "$WORK_DIR" > /dev/null

# --- Mutante race: propiedades LTL mutex y safe_update ---
MODEL="regression_workers_race.pml"
cp "$SCRIPT_DIR/$MODEL" .
echo "== Mutante race (NUM_WORKERS=${WORKERS} NUM_JOBS=${JOBS}) =="
spin -a -DNUM_WORKERS="${WORKERS}" -DNUM_JOBS="${JOBS}" "$MODEL"
gcc -o pan pan.c

for ltl_name in mutex safe_update; do
    echo "-- race / LTL: ${ltl_name} --"
    rm -f ./*.trail
    ./pan -a -N "${ltl_name}" | tee "$OUT_DIR/race_ltl_${ltl_name}.txt" || true
    expect_errors "$OUT_DIR/race_ltl_${ltl_name}.txt"
    replay_trail "$MODEL" "$OUT_DIR/race_ltl_${ltl_name}_trail.txt"
done

# --- Mutante deadlock: corrida de seguridad ---
MODEL="regression_workers_deadlock.pml"
cp "$SCRIPT_DIR/$MODEL" .
echo "== Mutante deadlock (NUM_WORKERS=${WORKERS} NUM_JOBS=${JOBS}) =="
rm -f ./*.trail pan pan.*
spin -a -DNUM_WORKERS="${WORKERS}" -DNUM_JOBS="${JOBS}" "$MODEL"
# -DNOCLAIM ignora las formulas ltl: chequeo puro de seguridad
# (invalid end states, assertion violations), igual que run_spin.sh.
gcc -DSAFETY -DNOCLAIM -o pan pan.c
./pan | tee "$OUT_DIR/deadlock_safety.txt" || true
expect_errors "$OUT_DIR/deadlock_safety.txt"
replay_trail "$MODEL" "$OUT_DIR/deadlock_safety_trail.txt"

popd > /dev/null

if [ "$UNDETECTED" -ne 0 ]; then
    echo "ERROR: al menos un mutante no fue detectado por Spin." >&2
    exit 1
fi

echo "Listo. Todos los mutantes fueron detectados. Resultados en $OUT_DIR"

#
# Alternativa con Docker (desde la raiz del repositorio):
#
#   docker run --rm -v "$PWD":/work -w /work debian:stable-slim bash -ec \
#     'apt-get update -qq && apt-get install -y -qq spin gcc libc6-dev && \
#      bash promela/run_spin_mutants.sh'
#
