package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	inputCSVPath  = "dataset/Hospital_Inpatient_Discharges_(SPARCS_De-Identified)__2022_20260913.csv"
	outputCSVPath = "dataset/SPARCS_2022_clean_go.csv"
)

var (
	firstIntegerRegex = regexp.MustCompile(`[0-9]+`)
	los120PlusRegex   = regexp.MustCompile(`^120[[:space:]]*\+`)
)

var outputColumns = []string{
	"Age Group",
	"Length of Stay",
	"LOS_120_plus",
	"Type of Admission",
	"APR Severity of Illness Code",
	"Severity_unknown",
	"APR MDC Description",
	"APR Medical Surgical Description",
	"Payment Typology 1",
	"Emergency Department Indicator",
	"Total Costs",
}

var requiredColumns = []string{
	"Age Group",
	"Length of Stay",
	"Type of Admission",
	"APR Severity of Illness Code",
	"APR MDC Description",
	"APR Medical Surgical Description",
	"Payment Typology 1",
	"Emergency Department Indicator",
	"Total Costs",
}

type CleaningStats struct {
	RowsRead                int
	RowsKept                int
	RowsDiscarded           int
	InvalidTotalCostRows    int
	InvalidLengthOfStayRows int
	LengthOfStay120PlusRows int
	UnknownSeverityRows     int
}

type CleanedRecord struct {
	AgeGroup                     string
	LengthOfStay                 int
	IsLengthOfStay120Plus        bool
	AdmissionType                string
	SeverityCode                 int
	IsSeverityUnknown            bool
	APRMajorDiagnosticCategory   string
	MedicalSurgicalCategory      string
	PrimaryPaymentType           string
	EmergencyDepartmentIndicator string
	TotalCost                    float64
}

func main() {
	inputFile, err := os.Open(inputCSVPath)
	if err != nil {
		fmt.Println("Error al abrir el dataset:", err)
		return
	}
	defer inputFile.Close()

	csvReader := csv.NewReader(inputFile)
	csvReader.FieldsPerRecord = -1

	header, err := csvReader.Read()
	if err != nil {
		fmt.Println("Error al leer la cabecera:", err)
		return
	}

	columnIndex := buildColumnIndex(header)

	if missingColumn := findMissingRequiredColumn(columnIndex); missingColumn != "" {
		fmt.Printf("No se encontró la columna obligatoria: %s\n", missingColumn)
		return
	}

	outputFile, err := os.Create(outputCSVPath)
	if err != nil {
		fmt.Println("Error al crear el archivo de salida:", err)
		return
	}
	defer outputFile.Close()

	csvWriter := csv.NewWriter(outputFile)
	defer csvWriter.Flush()

	if err := csvWriter.Write(outputColumns); err != nil {
		fmt.Println("Error al escribir la cabecera:", err)
		return
	}

	stats := CleaningStats{}

	for {
		rawRow, err := csvReader.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			fmt.Println("Error al leer una fila:", err)
			return
		}

		stats.RowsRead++

		cleanedRecord, discardReason, shouldKeep := cleanRecord(rawRow, columnIndex)

		if !shouldKeep {
			registerDiscard(&stats, discardReason)
			continue
		}

		if cleanedRecord.IsLengthOfStay120Plus {
			stats.LengthOfStay120PlusRows++
		}

		if cleanedRecord.IsSeverityUnknown {
			stats.UnknownSeverityRows++
		}

		if err := csvWriter.Write(cleanedRecord.toCSVRow()); err != nil {
			fmt.Println("Error al escribir una fila:", err)
			return
		}

		stats.RowsKept++
	}

	csvWriter.Flush()

	if err := csvWriter.Error(); err != nil {
		fmt.Println("Error durante la escritura:", err)
		return
	}

	printCleaningSummary(stats)
}

func cleanRecord(
	rawRow []string,
	columnIndex map[string]int,
) (CleanedRecord, string, bool) {

	valueOf := func(columnName string) string {
		position := columnIndex[columnName]

		if position < 0 || position >= len(rawRow) {
			return ""
		}

		return rawRow[position]
	}

	totalCost, validTotalCost := parseCurrency(valueOf("Total Costs"))

	if !validTotalCost || totalCost <= 0 {
		return CleanedRecord{}, "invalid_total_cost", false
	}

	lengthOfStay, is120Plus, validLengthOfStay := parseLengthOfStay(
		valueOf("Length of Stay"),
	)

	if !validLengthOfStay || lengthOfStay <= 0 {
		return CleanedRecord{}, "invalid_length_of_stay", false
	}

	severityCode, isSeverityUnknown := parseSeverityCode(
		valueOf("APR Severity of Illness Code"),
	)

	cleanedRecord := CleanedRecord{
		AgeGroup: normalizeCategory(
			valueOf("Age Group"),
		),
		LengthOfStay:          lengthOfStay,
		IsLengthOfStay120Plus: is120Plus,
		AdmissionType: normalizeCategory(
			valueOf("Type of Admission"),
		),
		SeverityCode:      severityCode,
		IsSeverityUnknown: isSeverityUnknown,
		APRMajorDiagnosticCategory: normalizeCategory(
			valueOf("APR MDC Description"),
		),
		MedicalSurgicalCategory: normalizeCategory(
			valueOf("APR Medical Surgical Description"),
		),
		PrimaryPaymentType: normalizeCategory(
			valueOf("Payment Typology 1"),
		),
		EmergencyDepartmentIndicator: normalizeCategory(
			valueOf("Emergency Department Indicator"),
		),
		TotalCost: totalCost,
	}

	return cleanedRecord, "", true
}

func parseCurrency(rawValue string) (float64, bool) {
	normalizedValue := strings.TrimSpace(rawValue)
	normalizedValue = strings.ReplaceAll(normalizedValue, ",", "")
	normalizedValue = strings.ReplaceAll(normalizedValue, "$", "")

	if normalizedValue == "" {
		return 0, false
	}

	parsedValue, err := strconv.ParseFloat(normalizedValue, 64)

	if err != nil {
		return 0, false
	}

	return parsedValue, true
}

func parseLengthOfStay(rawValue string) (int, bool, bool) {
	normalizedValue := strings.TrimSpace(rawValue)

	is120Plus := los120PlusRegex.MatchString(normalizedValue)

	numericPart := firstIntegerRegex.FindString(normalizedValue)

	if numericPart == "" {
		return 0, is120Plus, false
	}

	lengthOfStay, err := strconv.Atoi(numericPart)

	if err != nil {
		return 0, is120Plus, false
	}

	return lengthOfStay, is120Plus, true
}

func parseSeverityCode(rawValue string) (int, bool) {
	normalizedValue := strings.TrimSpace(rawValue)

	severityCode, err := strconv.Atoi(normalizedValue)

	if err != nil || severityCode < 1 || severityCode > 4 {
		return 0, true
	}

	return severityCode, false
}

func normalizeCategory(rawValue string) string {
	normalizedValue := strings.TrimSpace(rawValue)

	if normalizedValue == "" {
		return "Unknown"
	}

	return normalizedValue
}

func buildColumnIndex(header []string) map[string]int {
	columnIndex := make(map[string]int, len(header))

	for position, rawColumnName := range header {
		columnName := strings.TrimSpace(
			strings.TrimPrefix(rawColumnName, "\ufeff"),
		)

		columnIndex[columnName] = position
	}

	return columnIndex
}

func findMissingRequiredColumn(columnIndex map[string]int) string {
	for _, columnName := range requiredColumns {
		if _, exists := columnIndex[columnName]; !exists {
			return columnName
		}
	}

	return ""
}

func registerDiscard(stats *CleaningStats, reason string) {
	stats.RowsDiscarded++

	switch reason {
	case "invalid_total_cost":
		stats.InvalidTotalCostRows++

	case "invalid_length_of_stay":
		stats.InvalidLengthOfStayRows++
	}
}

func (record CleanedRecord) toCSVRow() []string {
	return []string{
		record.AgeGroup,
		strconv.Itoa(record.LengthOfStay),
		strconv.Itoa(boolToInt(record.IsLengthOfStay120Plus)),
		record.AdmissionType,
		strconv.Itoa(record.SeverityCode),
		strconv.Itoa(boolToInt(record.IsSeverityUnknown)),
		record.APRMajorDiagnosticCategory,
		record.MedicalSurgicalCategory,
		record.PrimaryPaymentType,
		record.EmergencyDepartmentIndicator,
		strconv.FormatFloat(record.TotalCost, 'f', -1, 64),
	}
}

func boolToInt(value bool) int {
	if value {
		return 1
	}

	return 0
}

func printCleaningSummary(stats CleaningStats) {
	fmt.Println()
	fmt.Println(" SPARCS 2022 - RESULTADO DE LIMPIEZA PC1")

	fmt.Printf("Registros leídos:                  %d\n", stats.RowsRead)
	fmt.Printf("Registros conservados:             %d\n", stats.RowsKept)
	fmt.Printf("Registros descartados:             %d\n", stats.RowsDiscarded)
	fmt.Printf("  - Total Costs inválido/no > 0:   %d\n", stats.InvalidTotalCostRows)
	fmt.Printf("  - Length of Stay inválido/no >0: %d\n", stats.InvalidLengthOfStayRows)
	fmt.Printf("Casos Length of Stay = 120+:       %d\n", stats.LengthOfStay120PlusRows)
	fmt.Printf("Casos de severidad desconocida:    %d\n", stats.UnknownSeverityRows)

	fmt.Println()
	fmt.Println("Dataset limpio generado en:")
	fmt.Println(outputCSVPath)
}
