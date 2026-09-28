package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
)

// defaultWorkerConfigurations es la lista de workers usada quando
// el usuario no especifica -workers.
var defaultWorkerConfigurations = []int{1, 2, 4, 8, 12, 16, 24, 32}

const (
	defaultEpochs       = 100
	defaultRuns         = 10
	defaultWarmup       = 1
	defaultTrimFraction = 0.1
	defaultOutDir       = "results"
)

var validModes = map[string]bool{
	"quick":       true,
	"benchmark":   true,
	"resources":   true,
	"cpu-profile": true,
	"clean":       true,
	"all":         true,
}

// Config agrupa todas las opciones configurables del CLI.
type Config struct {
	Mode         string
	Runs         int
	Warmup       int
	Epochs       int
	Workers      []int
	TrimFraction float64
	OutDir       string
	DatasetPath  string
}

func defaultConfig() Config {
	return Config{
		Mode:         "all",
		Runs:         defaultRuns,
		Warmup:       defaultWarmup,
		Epochs:       defaultEpochs,
		Workers:      append([]int(nil), defaultWorkerConfigurations...),
		TrimFraction: defaultTrimFraction,
		OutDir:       defaultOutDir,
		DatasetPath:  cleanDatasetPath,
	}
}

// parseConfig interpreta los argumentos de línea de comandos (sin
// incluir el nombre del binario, es decir os.Args[1:]).
//
// Compatibilidad retroactiva: si el primer argumento posicional es
// "cpu-profile" (sin guion), se interpreta como -mode=cpu-profile,
// igual que en versiones anteriores del CLI (`go run . cpu-profile`).
// Un -mode explícito después del positional puede reemplazarlo.
func parseConfig(args []string) (Config, error) {
	config := defaultConfig()

	if len(args) > 0 && args[0] == "cpu-profile" {
		config.Mode = "cpu-profile"
		args = args[1:]
	}

	flagSet := flag.NewFlagSet("pc2", flag.ContinueOnError)

	modeFlag := flagSet.String(
		"mode",
		config.Mode,
		"modo de ejecución: quick|benchmark|resources|cpu-profile|clean|all",
	)

	runsFlag := flagSet.Int(
		"runs",
		config.Runs,
		"número de ejecuciones por configuración en el benchmark",
	)

	warmupFlag := flagSet.Int(
		"warmup",
		config.Warmup,
		"ejecuciones de calentamiento descartadas antes de medir",
	)

	epochsFlag := flagSet.Int(
		"epochs",
		config.Epochs,
		"épocas de entrenamiento",
	)

	workersFlag := flagSet.String(
		"workers",
		joinInts(config.Workers),
		"lista de workers separada por comas, ej. 1,2,4,8",
	)

	trimFlag := flagSet.Float64(
		"trim",
		config.TrimFraction,
		"fracción a recortar POR LADO en la media recortada (ej. 0.1 = 10% inferior + 10% superior)",
	)

	outFlag := flagSet.String(
		"out",
		config.OutDir,
		"directorio de salida para los resultados persistidos",
	)

	datasetFlag := flagSet.String(
		"dataset",
		config.DatasetPath,
		"ruta al dataset limpio (CSV)",
	)

	if err := flagSet.Parse(args); err != nil {
		return Config{}, err
	}

	config.Mode = *modeFlag
	config.Runs = *runsFlag
	config.Warmup = *warmupFlag
	config.Epochs = *epochsFlag
	config.TrimFraction = *trimFlag
	config.OutDir = *outFlag
	config.DatasetPath = *datasetFlag

	workers, err := parseWorkerList(*workersFlag)

	if err != nil {
		return Config{}, err
	}

	config.Workers = workers

	if err := config.validate(); err != nil {
		return Config{}, err
	}

	return config, nil
}

func parseWorkerList(raw string) ([]int, error) {
	parts := strings.Split(raw, ",")
	workers := make([]int, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)

		if trimmed == "" {
			continue
		}

		value, err := strconv.Atoi(trimmed)

		if err != nil {
			return nil, fmt.Errorf("worker inválido %q: %w", trimmed, err)
		}

		if value < 1 {
			return nil, fmt.Errorf("worker inválido %q: debe ser >= 1", trimmed)
		}

		workers = append(workers, value)
	}

	if len(workers) == 0 {
		return nil, fmt.Errorf("la lista de workers (-workers) no puede estar vacía")
	}

	return workers, nil
}

func joinInts(values []int) string {
	parts := make([]string, len(values))

	for index, value := range values {
		parts[index] = strconv.Itoa(value)
	}

	return strings.Join(parts, ",")
}

func (config Config) validate() error {
	if !validModes[config.Mode] {
		return fmt.Errorf(
			"modo inválido %q: use quick|benchmark|resources|cpu-profile|clean|all",
			config.Mode,
		)
	}

	if config.Runs < 1 {
		return fmt.Errorf("-runs debe ser >= 1")
	}

	if config.Warmup < 0 {
		return fmt.Errorf("-warmup debe ser >= 0")
	}

	if config.Epochs < 1 {
		return fmt.Errorf("-epochs debe ser >= 1")
	}

	if config.TrimFraction < 0 || config.TrimFraction >= 0.5 {
		return fmt.Errorf("-trim debe estar en el rango [0, 0.5)")
	}

	if config.OutDir == "" {
		return fmt.Errorf("-out no puede estar vacío")
	}

	if config.DatasetPath == "" {
		return fmt.Errorf("-dataset no puede estar vacío")
	}

	return nil
}
