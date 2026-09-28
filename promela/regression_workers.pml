#ifndef NUM_WORKERS
#define NUM_WORKERS 3
#endif

#ifndef NUM_JOBS
#define NUM_JOBS 4
#endif

#define STOP 255

/*
    Modelo simplificado de la arquitectura concurrente
    utilizada para el entrenamiento de Regresión Lineal.

    En Go:
        Coordinator
             |
          jobs chan
             |
        Worker Pool
             |
        results chan
             |
          Reduce
             |
      Update Weights

    El objetivo del modelo NO es representar los
    2 millones de registros ni las operaciones
    matemáticas de Gradient Descent.

    El objetivo es verificar la sincronización.

    NUM_WORKERS y NUM_JOBS son configurables en tiempo de
    compilación con -D (ver promela/run_spin.sh y
    run_spin.ps1), p. ej.:

        spin -a -DNUM_WORKERS=2 -DNUM_JOBS=4 regression_workers.pml
        spin -a -DNUM_WORKERS=4 -DNUM_JOBS=8 regression_workers.pml
*/

chan jobs = [NUM_JOBS + NUM_WORKERS] of { byte };
chan results = [NUM_JOBS] of { byte };

/*
    processed[job] indica si el job fue procesado.
*/
byte processed[NUM_JOBS];

/*
    received[job] indica si el resultado del job
    fue recibido por el coordinador.
*/
byte received[NUM_JOBS];

byte result_count = 0;

bool weights_updated = false;

/*
    in_update cuenta cuántos procesos están, en un
    instante dado, dentro de la sección crítica que
    actualiza los pesos globales (equivalente a la
    región entre wg.Wait() y el fin del epoch en
    concurrent.go, donde SOLO el Coordinator escribe
    model.Weights/model.Bias).

    updating es verdadero mientras esa sección crítica
    está activa. Ningún Worker participa de ella: los
    Workers solo leen los pesos (antes del epoch) y
    producen gradientes parciales privados, por lo que
    no existe una condición de carrera real que exija un
    Mutex explícito en Go. El modelo verifica formalmente
    esa propiedad (mutex/exclusión mutua) en vez de
    asumirla.
*/
byte in_update = 0;
bool updating = false;

/*
    done se activa cuando el Coordinator terminó todo el
    ciclo (reduce + update). Se usa para la propiedad LTL
    de terminación (ausencia de deadlock/livelock).
*/
bool done = false;

/*
    Cada Worker consume jobs del canal.

    Los Workers NO actualizan los pesos globales.
    Solo producen un resultado parcial.
*/
proctype Worker(byte worker_id)
{
    byte job_id;

    do
    :: jobs?job_id ->

        if
        :: job_id == STOP ->

            break

        :: else ->

            assert(job_id < NUM_JOBS);

            /*
                Un mismo job no debe procesarse
                más de una vez.
            */
            atomic {
                assert(processed[job_id] == 0);
                processed[job_id] = 1;
            }

            /*
                Representa el envío del gradiente
                parcial calculado por este worker.
            */
            results!job_id
        fi
    od
}


/*
    El proceso init representa al Coordinator.
*/
init
{
    byte i;
    byte result_id;

    /*
        Crear Worker Pool.
    */
    atomic {

        i = 0;

        do
        :: i < NUM_WORKERS ->

            run Worker(i);

            i++

        :: else ->
            break
        od
    }


    /*
        Enviar todos los jobs.
    */
    i = 0;

    do
    :: i < NUM_JOBS ->

        jobs!i;

        i++

    :: else ->
        break
    od


    /*
        Enviar señal STOP a cada Worker.
    */
    i = 0;

    do
    :: i < NUM_WORKERS ->

        jobs!STOP;

        i++

    :: else ->
        break
    od


    /*
        Reducer:

        El Coordinator recibe exactamente
        un resultado por cada job.
    */
    i = 0;

    do
    :: i < NUM_JOBS ->

        results?result_id;

        assert(result_id < NUM_JOBS);

        /*
            El resultado de cada job solo
            debe recibirse una vez.
        */
        assert(received[result_id] == 0);

        received[result_id] = 1;

        result_count++;

        i++

    :: else ->
        break
    od


    /*
        Antes de actualizar los pesos,
        deben haberse recibido TODOS
        los gradientes parciales.
    */
    assert(result_count == NUM_JOBS);


    /*
        Verificar que todos los jobs fueron
        procesados exactamente una vez y
        que todos sus resultados llegaron.
    */
    i = 0;

    do
    :: i < NUM_JOBS ->

        assert(processed[i] == 1);
        assert(received[i] == 1);

        i++

    :: else ->
        break
    od


    /*
        Únicamente el Coordinator llega a
        este punto y actualiza los pesos.

        Los Workers nunca modifican los
        pesos globales directamente.

        Entrar/salir de la sección crítica de
        actualización se modela de forma atómica
        para poder verificar exclusión mutua
        (mutex) sobre in_update, análogo a la
        región de concurrent.go entre wg.Wait()
        y el fin del epoch.
    */
    atomic {
        in_update++;
        updating = true;
    }

    weights_updated = true;

    assert(weights_updated);
    assert(result_count == NUM_JOBS);

    atomic {
        updating = false;
        in_update--;
    }

    done = true;
}

/*
    Propiedades LTL
    ================

    safe_update: cada vez que el Coordinator está
    actualizando los pesos (updating), YA se recibieron
    todos los gradientes parciales (result_count ==
    NUM_JOBS). Equivale a "nunca se actualiza con datos
    incompletos", el invariante central del wg.Wait()
    antes de la actualización en concurrent.go.

    termination: eventualmente el Coordinator termina
    todo el ciclo (done se vuelve verdadero). Ausencia de
    deadlock/livelock para este modelo.

    mutex: en todo momento, a lo sumo un proceso está
    dentro de la sección crítica de actualización de
    pesos (in_update <= 1). Verifica formalmente que no
    hace falta un Mutex adicional en Go: la exclusión ya
    está garantizada por diseño (solo el Coordinator
    entra a esa sección, después de wg.Wait()).

    Ejecutar con -N <nombre> selecciona una sola fórmula
    por corrida de pan (ver run_spin.sh / run_spin.ps1).
*/
ltl safe_update { [] (updating -> result_count == NUM_JOBS) }
ltl termination { <> done }
ltl mutex { [] (in_update <= 1) }
