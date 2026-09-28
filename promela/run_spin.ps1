<#
.SYNOPSIS
    Ejecuta la verificación formal (Spin + gcc/pan) del modelo
    promela/regression_workers.pml: seguridad (deadlock / invalid
    end states / assertion violations) y cada propiedad LTL por
    separado (-N <nombre>).

.DESCRIPTION
    Requiere `spin` y `gcc` (p. ej. vía MSYS2/MinGW) instalados y
    en el PATH. NO están instalados en el entorno donde se generó
    este script; el equipo debe ejecutarlo localmente o vía Docker
    (ver comentario al final) para producir la evidencia real.

.PARAMETER Workers
    Valor de NUM_WORKERS para esta corrida. Default: 3.

.PARAMETER Jobs
    Valor de NUM_JOBS para esta corrida. Default: 4.

.EXAMPLE
    ./run_spin.ps1
    ./run_spin.ps1 -Workers 2 -Jobs 4
    ./run_spin.ps1 -Workers 4 -Jobs 4
#>

param(
    [int]$Workers = 3,
    [int]$Jobs = 4
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$Model = Join-Path $ScriptDir "regression_workers.pml"
$OutDir = Join-Path $ScriptDir "..\results\promela"
$WorkDir = Join-Path ([System.IO.Path]::GetTempPath()) ("spin_" + [System.Guid]::NewGuid().ToString("N"))

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
New-Item -ItemType Directory -Force -Path $WorkDir | Out-Null

$Prefix = "w${Workers}_j${Jobs}"

# $ErrorActionPreference no detiene el script cuando falla un
# ejecutable nativo; se revisa $LASTEXITCODE explícitamente.
function Assert-LastExitCode([string]$Step) {
    if ($LASTEXITCODE -ne 0) {
        throw "$Step falló con código de salida $LASTEXITCODE"
    }
}

Write-Host "== Verificando NUM_WORKERS=$Workers NUM_JOBS=$Jobs =="

Push-Location $WorkDir
try {
    # 1) Generar el verificador C a partir del modelo, con las
    #    defines de la variante actual.
    spin -a -DNUM_WORKERS=$Workers -DNUM_JOBS=$Jobs $Model
    Assert-LastExitCode "spin -a"

    # 2) Compilar y correr la verificación de seguridad
    #    (deadlock, invalid end states, assertion violations).
    #    -DNOCLAIM ignora las fórmulas ltl del modelo para que
    #    esta corrida sea un chequeo puro de seguridad.
    gcc -DSAFETY -DNOCLAIM -o pan.exe pan.c
    Assert-LastExitCode "gcc (seguridad)"
    & .\pan.exe | Tee-Object -FilePath (Join-Path $OutDir "${Prefix}_safety.txt")
    Assert-LastExitCode "pan (seguridad)"

    # 3) Compilar (sin -DSAFETY) y correr cada propiedad LTL
    #    por separado con -N <nombre>.
    gcc -o pan.exe pan.c
    Assert-LastExitCode "gcc (LTL)"

    foreach ($ltlName in @("safe_update", "termination", "mutex")) {
        Write-Host "-- LTL: $ltlName --"
        & .\pan.exe -a -N $ltlName | Tee-Object -FilePath (Join-Path $OutDir "${Prefix}_ltl_${ltlName}.txt")
        Assert-LastExitCode "pan (LTL $ltlName)"
    }
}
finally {
    Pop-Location
    Remove-Item -Recurse -Force $WorkDir -ErrorAction SilentlyContinue
}

Write-Host "Listo. Resultados en $OutDir"

<#
Alternativa con Docker (si spin/gcc no están disponibles
localmente vía MSYS2/MinGW, p. ej. en este entorno de
desarrollo):

    docker run --rm -v ${PWD}:/work -w /work `
      -e WORKERS=3 -e JOBS=4 `
      <imagen-con-spin-y-gcc> `
      bash -c 'promela/run_spin.sh "$WORKERS" "$JOBS"'

No existe todavía una imagen oficial del equipo; se puede
construir una mínima con `apt-get install -y spin gcc` (Debian/
Ubuntu) o usar una imagen pública con Spin preinstalado. El
script .sh dentro del contenedor reutiliza la misma lógica que
este .ps1.
#>
