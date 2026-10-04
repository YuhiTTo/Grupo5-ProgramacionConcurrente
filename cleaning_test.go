package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCleaning_MissingInputReturnsError(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "clean.csv")

	err := runCleaning(filepath.Join(dir, "no-existe.csv"), output)

	if err == nil {
		t.Fatal("se esperaba un error cuando el dataset de entrada no existe")
	}

	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Errorf("no debería existir el archivo de salida, stat err = %v", statErr)
	}
}

const rawCleaningHeader = "Age Group,Length of Stay,Type of Admission,APR Severity of Illness Code," +
	"APR MDC Description,APR Medical Surgical Description,Payment Typology 1," +
	"Emergency Department Indicator,Total Costs\n"

func writeRawFixture(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "raw.csv")

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("no se pudo escribir el fixture: %v", err)
	}

	return path
}

func TestRunCleaning_SuccessWritesFinalFileAtomically(t *testing.T) {
	input := writeRawFixture(t, rawCleaningHeader+
		"30 to 49,3,Emergency,2,Diseases of the Ear,Medical,Medicare,Y,\"$1,234.50\"\n"+
		"70 or Older,120 +,Urgent,4,Diseases of the Ear,Surgical,Medicaid,N,$9000.00\n"+
		"0 to 17,0,Elective,1,Diseases of the Ear,Medical,Self-Pay,N,$100.00\n")
	output := filepath.Join(t.TempDir(), "clean.csv")

	if err := runCleaning(input, output); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("no se pudo leer la salida: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")

	// header + 2 kept rows (the third has Length of Stay 0 and is discarded)
	if len(lines) != 3 {
		t.Fatalf("se esperaban 3 líneas (cabecera + 2 filas), se obtuvieron %d: %q", len(lines), lines)
	}

	if lines[1] != "30 to 49,3,0,Emergency,2,0,Diseases of the Ear,Medical,Medicare,Y,1234.5" {
		t.Errorf("fila limpia inesperada: %q", lines[1])
	}

	if _, statErr := os.Stat(output + ".part"); !os.IsNotExist(statErr) {
		t.Errorf("no debería quedar .part, stat err = %v", statErr)
	}
}

func TestRunCleaning_FailureLeavesNoFinalFile(t *testing.T) {
	// The unterminated quote makes the CSV reader fail mid-file.
	input := writeRawFixture(t, rawCleaningHeader+
		"30 to 49,3,Emergency,2,Diseases of the Ear,Medical,Medicare,Y,$100.00\n"+
		"30 to 49,3,Emergency,2,\"broken,Medical,Medicare,Y,$100.00\n")
	output := filepath.Join(t.TempDir(), "clean.csv")

	if err := runCleaning(input, output); err == nil {
		t.Fatal("se esperaba un error por CSV malformado")
	}

	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Errorf("no debería existir el archivo final, stat err = %v", statErr)
	}

	if _, statErr := os.Stat(output + ".part"); !os.IsNotExist(statErr) {
		t.Errorf("no debería quedar .part, stat err = %v", statErr)
	}
}

func TestRunCleaning_FailurePreservesExistingFinalFile(t *testing.T) {
	input := writeRawFixture(t, rawCleaningHeader+
		"30 to 49,3,Emergency,2,\"broken,Medical,Medicare,Y,$100.00\n")
	output := filepath.Join(t.TempDir(), "clean.csv")

	if err := os.WriteFile(output, []byte("previous"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := runCleaning(input, output); err == nil {
		t.Fatal("se esperaba un error por CSV malformado")
	}

	content, _ := os.ReadFile(output)
	if string(content) != "previous" {
		t.Errorf("el archivo final previo fue modificado: %q", content)
	}
}
