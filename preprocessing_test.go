package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const (
	circulatoryMDC  = "DISEASES AND DISORDERS OF THE CIRCULATORY SYSTEM"
	respiratoryMDC  = "DISEASES AND DISORDERS OF THE RESPIRATORY SYSTEM"
	cleanTestHeader = "Age Group,Length of Stay,LOS_120_plus,Type of Admission," +
		"APR Severity of Illness Code,Severity_unknown,APR MDC Description," +
		"APR Medical Surgical Description,Payment Typology 1," +
		"Emergency Department Indicator,Total Costs\n"
)

// writeCleanFixture writes a tiny cleaned dataset whose categories include
// every reference category required by buildFeatureSchema.
func writeCleanFixture(t *testing.T, extraRows string) string {
	t.Helper()

	content := cleanTestHeader +
		"0 to 17,2,0,Emergency,1,0," + circulatoryMDC + ",Medical,Medicare,Y,100\n" +
		"30 to 49,5,0,Elective,3,0," + respiratoryMDC + ",Surgical,Medicaid,N,250.5\n" +
		"70 or Older,120,1,Emergency,4,0," + circulatoryMDC + ",Medical,Medicare,N,9000\n" +
		extraRows

	path := filepath.Join(t.TempDir(), "clean.csv")

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("no se pudo escribir el fixture: %v", err)
	}

	return path
}

func TestAnalyzePreprocessingSchemaAndFeatureSchema(t *testing.T) {
	path := writeCleanFixture(t, "")

	summary, err := analyzePreprocessingSchema(path)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	wantAges := []string{"0 to 17", "30 to 49", "70 or Older"}
	if !reflect.DeepEqual(summary.AgeGroups, wantAges) {
		t.Errorf("AgeGroups = %v, se esperaba %v", summary.AgeGroups, wantAges)
	}

	// 5 direct features + 2 + 1 + 1 + 1 + 1 dummies (k-1 per categorical).
	if summary.FinalFeatureCount != 11 {
		t.Errorf("FinalFeatureCount = %d, se esperaba 11", summary.FinalFeatureCount)
	}

	schema, err := buildFeatureSchema(summary)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if schema.FeatureCount != 11 {
		t.Errorf("FeatureCount = %d, se esperaba 11", schema.FeatureCount)
	}

	wantAgeIndex := map[string]int{"0 to 17": -1, "30 to 49": 5, "70 or Older": 6}
	if !reflect.DeepEqual(schema.AgeGroupIndex, wantAgeIndex) {
		t.Errorf("AgeGroupIndex = %v, se esperaba %v", schema.AgeGroupIndex, wantAgeIndex)
	}

	wantAdmission := map[string]int{"Elective": 7, "Emergency": -1}
	if !reflect.DeepEqual(schema.AdmissionTypeIndex, wantAdmission) {
		t.Errorf("AdmissionTypeIndex = %v, se esperaba %v", schema.AdmissionTypeIndex, wantAdmission)
	}
}

func TestBuildFeatureSchemaRequiresReferenceCategory(t *testing.T) {
	summary := PreprocessingSummary{
		AgeGroups:             []string{"30 to 49"}, // no "0 to 17" reference
		AdmissionTypes:        []string{"Emergency"},
		MajorDiagnosticGroups: []string{circulatoryMDC},
		MedicalSurgicalGroups: []string{"Medical"},
		PaymentTypes:          []string{"Medicare"},
	}

	if _, err := buildFeatureSchema(summary); err == nil {
		t.Fatal("se esperaba un error por falta de la categoría de referencia")
	}
}

func TestLoadAndSplitDatasetEncodesCategoricalDummies(t *testing.T) {
	path := writeCleanFixture(t, "")

	summary, err := analyzePreprocessingSchema(path)
	if err != nil {
		t.Fatal(err)
	}

	schema, err := buildFeatureSchema(summary)
	if err != nil {
		t.Fatal(err)
	}

	// trainRatio 1.0: rand.Float64() is always < 1, so every row is train.
	train, test, err := loadAndSplitDataset(path, schema, 3, 1.0, 42)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if len(train) != 3 || len(test) != 0 {
		t.Fatalf("train=%d test=%d, se esperaba 3/0", len(train), len(test))
	}

	// Row 1: every categorical is its reference category -> all -1.
	first := train[0]
	if first.AgeFeature != -1 || first.AdmissionFeature != -1 || first.MDCFeature != -1 ||
		first.MedicalSurgicalFeature != -1 || first.PaymentFeature != -1 {
		t.Errorf("fila 1: se esperaban todas las categóricas en -1, se obtuvo %+v", first)
	}

	if first.EmergencyDepartment != 1 || first.LengthOfStay != 2 || first.Severity != 1 || first.Target != 100 {
		t.Errorf("fila 1: features directas inesperadas: %+v", first)
	}

	// Row 2: all categoricals active.
	second := train[1]
	if second.AgeFeature != 5 || second.AdmissionFeature != 7 || second.MDCFeature != 8 ||
		second.MedicalSurgicalFeature != 9 || second.PaymentFeature != 10 {
		t.Errorf("fila 2: índices dummy inesperados: %+v", second)
	}

	if second.EmergencyDepartment != 0 || second.Target != 250.5 {
		t.Errorf("fila 2: features directas inesperadas: %+v", second)
	}

	// Row 3: 120+ stay flagged, age dummy for "70 or Older".
	third := train[2]
	if third.AgeFeature != 6 || third.LOS120Plus != 1 || third.LengthOfStay != 120 {
		t.Errorf("fila 3: valores inesperados: %+v", third)
	}
}

func TestLoadAndSplitDatasetRejectsUnknownCategory(t *testing.T) {
	path := writeCleanFixture(t, "")

	summary, err := analyzePreprocessingSchema(path)
	if err != nil {
		t.Fatal(err)
	}

	schema, err := buildFeatureSchema(summary)
	if err != nil {
		t.Fatal(err)
	}

	// A dataset with a category absent from the schema must be rejected.
	other := writeCleanFixture(t,
		"40 to 59,4,0,Emergency,2,0,"+circulatoryMDC+",Medical,Medicare,Y,300\n")

	if _, _, err := loadAndSplitDataset(other, schema, 4, 1.0, 42); err == nil {
		t.Fatal("se esperaba un error por categoría desconocida")
	}
}

func writeRawText(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "custom.csv")

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("no se pudo escribir el fixture: %v", err)
	}

	return path
}

func TestAnalyzePreprocessingSchemaRejectsMissingColumn(t *testing.T) {
	path := writeRawText(t, "Age Group,Length of Stay\n0 to 17,2\n")

	_, err := analyzePreprocessingSchema(path)

	if err == nil {
		t.Fatal("se esperaba un error por columnas obligatorias ausentes")
	}

	if !strings.Contains(err.Error(), "columna") {
		t.Errorf("el error debería mencionar la columna faltante: %v", err)
	}
}

func TestAnalyzePreprocessingSchemaRejectsShortRow(t *testing.T) {
	path := writeCleanFixture(t, "0 to 17,2,0\n")

	_, err := analyzePreprocessingSchema(path)

	if err == nil {
		t.Fatal("se esperaba un error por fila con menos columnas que la cabecera")
	}
}

func TestLoadAndSplitDatasetRejectsMissingColumnAndShortRow(t *testing.T) {
	good := writeCleanFixture(t, "")

	summary, err := analyzePreprocessingSchema(good)
	if err != nil {
		t.Fatal(err)
	}

	schema, err := buildFeatureSchema(summary)
	if err != nil {
		t.Fatal(err)
	}

	missing := writeRawText(t, "Age Group,Length of Stay\n0 to 17,2\n")

	if _, _, err := loadAndSplitDataset(missing, schema, 1, 1.0, 42); err == nil {
		t.Error("se esperaba un error por columnas obligatorias ausentes")
	}

	short := writeCleanFixture(t, "0 to 17,2,0\n")

	if _, _, err := loadAndSplitDataset(short, schema, 4, 1.0, 42); err == nil {
		t.Error("se esperaba un error por fila corta")
	}
}
