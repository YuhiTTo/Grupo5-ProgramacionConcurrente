package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sort"
	"strconv"
)

type PreprocessingSummary struct {
	AgeGroups             []string
	AdmissionTypes        []string
	MajorDiagnosticGroups []string
	MedicalSurgicalGroups []string
	PaymentTypes          []string
	EmergencyIndicators   []string
	FinalFeatureCount     int
}

func analyzePreprocessingSchema(filePath string) (PreprocessingSummary, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return PreprocessingSummary{}, fmt.Errorf(
			"no se pudo abrir el dataset: %w",
			err,
		)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err != nil {
		return PreprocessingSummary{}, fmt.Errorf(
			"no se pudo leer la cabecera: %w",
			err,
		)
	}

	columnIndex, minRowLength, err := validateCleanHeader(header)
	if err != nil {
		return PreprocessingSummary{}, err
	}

	ageGroups := make(map[string]struct{})
	admissionTypes := make(map[string]struct{})
	majorDiagnosticGroups := make(map[string]struct{})
	medicalSurgicalGroups := make(map[string]struct{})
	paymentTypes := make(map[string]struct{})
	emergencyIndicators := make(map[string]struct{})

	for {
		row, err := reader.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			return PreprocessingSummary{}, fmt.Errorf(
				"error leyendo el dataset: %w",
				err,
			)
		}

		if err := validateRowLength(row, minRowLength); err != nil {
			return PreprocessingSummary{}, err
		}

		ageGroups[row[columnIndex["Age Group"]]] = struct{}{}
		admissionTypes[row[columnIndex["Type of Admission"]]] = struct{}{}
		majorDiagnosticGroups[row[columnIndex["APR MDC Description"]]] = struct{}{}
		medicalSurgicalGroups[row[columnIndex["APR Medical Surgical Description"]]] = struct{}{}
		paymentTypes[row[columnIndex["Payment Typology 1"]]] = struct{}{}
		emergencyIndicators[row[columnIndex["Emergency Department Indicator"]]] = struct{}{}
	}

	summary := PreprocessingSummary{
		AgeGroups:             sortedKeys(ageGroups),
		AdmissionTypes:        sortedKeys(admissionTypes),
		MajorDiagnosticGroups: sortedKeys(majorDiagnosticGroups),
		MedicalSurgicalGroups: sortedKeys(medicalSurgicalGroups),
		PaymentTypes:          sortedKeys(paymentTypes),
		EmergencyIndicators:   sortedKeys(emergencyIndicators),
	}

	// Variables numéricas directas:
	//
	// Length of Stay
	// LOS_120_plus
	// APR Severity of Illness Code
	// Severity_unknown
	//
	// = 4
	//
	// Emergency Department Indicator
	// = 1 binaria
	//
	// Para cada categórica nominal usamos k-1 columnas dummy.
	summary.FinalFeatureCount =
		4 +
			1 +
			(len(summary.AgeGroups) - 1) +
			(len(summary.AdmissionTypes) - 1) +
			(len(summary.MajorDiagnosticGroups) - 1) +
			(len(summary.MedicalSurgicalGroups) - 1) +
			(len(summary.PaymentTypes) - 1)

	return summary, nil
}

// validateCleanHeader verifica que la cabecera del dataset limpio
// contenga todas las columnas esperadas y devuelve su índice junto con
// la longitud mínima que debe tener cada fila para poder leerlas.
func validateCleanHeader(header []string) (map[string]int, int, error) {
	columnIndex := buildColumnIndex(header)

	for _, columnName := range outputColumns {
		if _, exists := columnIndex[columnName]; !exists {
			return nil, 0, fmt.Errorf(
				"el dataset no contiene la columna obligatoria %q",
				columnName,
			)
		}
	}

	minRowLength := 0

	for _, columnName := range outputColumns {
		if position := columnIndex[columnName] + 1; position > minRowLength {
			minRowLength = position
		}
	}

	return columnIndex, minRowLength, nil
}

// validateRowLength evita un panic por índice fuera de rango cuando una
// fila tiene menos columnas que las requeridas.
func validateRowLength(row []string, minRowLength int) error {
	if len(row) < minRowLength {
		return fmt.Errorf(
			"fila con %d columnas; se esperaban al menos %d",
			len(row),
			minRowLength,
		)
	}

	return nil
}

func sortedKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))

	for value := range values {
		result = append(result, value)
	}

	sort.Strings(result)

	return result
}

func printCategorySummary(name string, values []string) {
	fmt.Printf("\n%s (%d categorías)\n", name, len(values))

	for _, value := range values {
		fmt.Printf("  - %s\n", value)
	}
}

type FeatureSchema struct {
	FeatureCount int

	AgeGroupIndex        map[string]int
	AdmissionTypeIndex   map[string]int
	MDCIndex             map[string]int
	MedicalSurgicalIndex map[string]int
	PaymentTypeIndex     map[string]int
}

type Sample struct {
	LengthOfStay        float64
	LOS120Plus          float64
	Severity            float64
	SeverityUnknown     float64
	EmergencyDepartment float64

	AgeFeature             int
	AdmissionFeature       int
	MDCFeature             int
	MedicalSurgicalFeature int
	PaymentFeature         int

	Target float64
}

func buildFeatureSchema(
	summary PreprocessingSummary,
) (FeatureSchema, error) {

	const directFeatureCount = 5

	nextFeature := directFeatureCount

	schema := FeatureSchema{
		AgeGroupIndex:        make(map[string]int),
		AdmissionTypeIndex:   make(map[string]int),
		MDCIndex:             make(map[string]int),
		MedicalSurgicalIndex: make(map[string]int),
		PaymentTypeIndex:     make(map[string]int),
	}

	var err error

	// ---------------------------------------------------------
	// Age Group
	// Referencia: 0 to 17
	// ---------------------------------------------------------
	nextFeature, err = addOneHotFeatures(
		schema.AgeGroupIndex,
		summary.AgeGroups,
		"0 to 17",
		nextFeature,
	)

	if err != nil {
		return FeatureSchema{}, err
	}

	// ---------------------------------------------------------
	// Type of Admission
	// Referencia: Emergency
	// ---------------------------------------------------------
	nextFeature, err = addOneHotFeatures(
		schema.AdmissionTypeIndex,
		summary.AdmissionTypes,
		"Emergency",
		nextFeature,
	)

	if err != nil {
		return FeatureSchema{}, err
	}

	// ---------------------------------------------------------
	// APR MDC
	// Referencia: sistema circulatorio
	// ---------------------------------------------------------
	nextFeature, err = addOneHotFeatures(
		schema.MDCIndex,
		summary.MajorDiagnosticGroups,
		"DISEASES AND DISORDERS OF THE CIRCULATORY SYSTEM",
		nextFeature,
	)

	if err != nil {
		return FeatureSchema{}, err
	}

	// ---------------------------------------------------------
	// Medical / Surgical
	// Referencia: Medical
	// ---------------------------------------------------------
	nextFeature, err = addOneHotFeatures(
		schema.MedicalSurgicalIndex,
		summary.MedicalSurgicalGroups,
		"Medical",
		nextFeature,
	)

	if err != nil {
		return FeatureSchema{}, err
	}

	// ---------------------------------------------------------
	// Payment Typology
	// Referencia: Medicare
	// ---------------------------------------------------------
	nextFeature, err = addOneHotFeatures(
		schema.PaymentTypeIndex,
		summary.PaymentTypes,
		"Medicare",
		nextFeature,
	)

	if err != nil {
		return FeatureSchema{}, err
	}

	schema.FeatureCount = nextFeature

	return schema, nil
}

func addOneHotFeatures(
	destination map[string]int,
	categories []string,
	referenceCategory string,
	startIndex int,
) (int, error) {

	referenceFound := false
	nextIndex := startIndex

	for _, category := range categories {

		if category == referenceCategory {
			// -1 significa:
			// categoría de referencia, por lo que todas
			// sus dummies son cero.
			destination[category] = -1
			referenceFound = true
			continue
		}

		destination[category] = nextIndex
		nextIndex++
	}

	if !referenceFound {
		return 0, fmt.Errorf(
			"no se encontró la categoría de referencia %q",
			referenceCategory,
		)
	}

	return nextIndex, nil
}

func loadAndSplitDataset(
	filePath string,
	schema FeatureSchema,
	expectedRows int,
	trainRatio float64,
	randomSeed int64,
) ([]Sample, []Sample, error) {

	file, err := os.Open(filePath)

	if err != nil {
		return nil, nil, fmt.Errorf(
			"no se pudo abrir el dataset: %w",
			err,
		)
	}

	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	header, err := reader.Read()

	if err != nil {
		return nil, nil, fmt.Errorf(
			"no se pudo leer la cabecera: %w",
			err,
		)
	}

	columnIndex, minRowLength, err := validateCleanHeader(header)

	if err != nil {
		return nil, nil, err
	}

	expectedTrainRows := int(
		float64(expectedRows) * trainRatio,
	)

	expectedTestRows :=
		expectedRows - expectedTrainRows

	trainData := make(
		[]Sample,
		0,
		expectedTrainRows,
	)

	testData := make(
		[]Sample,
		0,
		expectedTestRows,
	)

	randomGenerator := rand.New(
		rand.NewSource(randomSeed),
	)

	for {
		row, err := reader.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, nil, fmt.Errorf(
				"error leyendo el dataset: %w",
				err,
			)
		}

		if err := validateRowLength(row, minRowLength); err != nil {
			return nil, nil, err
		}

		sample, err := parseSample(
			row,
			columnIndex,
			schema,
		)

		if err != nil {
			return nil, nil, err
		}

		if randomGenerator.Float64() < trainRatio {
			trainData = append(
				trainData,
				sample,
			)
		} else {
			testData = append(
				testData,
				sample,
			)
		}
	}

	return trainData, testData, nil
}

func parseSample(
	row []string,
	columnIndex map[string]int,
	schema FeatureSchema,
) (Sample, error) {

	valueOf := func(columnName string) string {
		return row[columnIndex[columnName]]
	}

	lengthOfStay, err := strconv.ParseFloat(
		valueOf("Length of Stay"),
		64,
	)

	if err != nil {
		return Sample{}, fmt.Errorf(
			"Length of Stay inválido: %w",
			err,
		)
	}

	los120Plus, err := strconv.ParseFloat(
		valueOf("LOS_120_plus"),
		64,
	)

	if err != nil {
		return Sample{}, fmt.Errorf(
			"LOS_120_plus inválido: %w",
			err,
		)
	}

	severity, err := strconv.ParseFloat(
		valueOf("APR Severity of Illness Code"),
		64,
	)

	if err != nil {
		return Sample{}, fmt.Errorf(
			"Severity inválido: %w",
			err,
		)
	}

	severityUnknown, err := strconv.ParseFloat(
		valueOf("Severity_unknown"),
		64,
	)

	if err != nil {
		return Sample{}, fmt.Errorf(
			"Severity_unknown inválido: %w",
			err,
		)
	}

	totalCost, err := strconv.ParseFloat(
		valueOf("Total Costs"),
		64,
	)

	if err != nil {
		return Sample{}, fmt.Errorf(
			"Total Costs inválido: %w",
			err,
		)
	}

	emergencyDepartment := 0.0

	switch valueOf("Emergency Department Indicator") {

	case "N":
		emergencyDepartment = 0

	case "Y":
		emergencyDepartment = 1

	default:
		return Sample{}, fmt.Errorf(
			"Emergency Department Indicator desconocido: %q",
			valueOf("Emergency Department Indicator"),
		)
	}

	ageFeature, err := featureIndex(
		schema.AgeGroupIndex,
		valueOf("Age Group"),
		"Age Group",
	)

	if err != nil {
		return Sample{}, err
	}

	admissionFeature, err := featureIndex(
		schema.AdmissionTypeIndex,
		valueOf("Type of Admission"),
		"Type of Admission",
	)

	if err != nil {
		return Sample{}, err
	}

	mdcFeature, err := featureIndex(
		schema.MDCIndex,
		valueOf("APR MDC Description"),
		"APR MDC Description",
	)

	if err != nil {
		return Sample{}, err
	}

	medicalSurgicalFeature, err := featureIndex(
		schema.MedicalSurgicalIndex,
		valueOf("APR Medical Surgical Description"),
		"APR Medical Surgical Description",
	)

	if err != nil {
		return Sample{}, err
	}

	paymentFeature, err := featureIndex(
		schema.PaymentTypeIndex,
		valueOf("Payment Typology 1"),
		"Payment Typology 1",
	)

	if err != nil {
		return Sample{}, err
	}

	return Sample{
		LengthOfStay:        lengthOfStay,
		LOS120Plus:          los120Plus,
		Severity:            severity,
		SeverityUnknown:     severityUnknown,
		EmergencyDepartment: emergencyDepartment,

		AgeFeature:             ageFeature,
		AdmissionFeature:       admissionFeature,
		MDCFeature:             mdcFeature,
		MedicalSurgicalFeature: medicalSurgicalFeature,
		PaymentFeature:         paymentFeature,

		Target: totalCost,
	}, nil
}

func featureIndex(
	indexMap map[string]int,
	category string,
	columnName string,
) (int, error) {

	index, exists := indexMap[category]

	if !exists {
		return 0, fmt.Errorf(
			"categoría desconocida en %s: %q",
			columnName,
			category,
		)
	}

	return index, nil
}
