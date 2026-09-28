package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
)

type DatasetInfo struct {
	RowCount int
	Columns  []string
}

func inspectCleanDataset(filePath string) (DatasetInfo, error) {
	file, err := os.Open(filePath)

	if err != nil {
		return DatasetInfo{}, fmt.Errorf(
			"no se pudo abrir el dataset limpio: %w",
			err,
		)
	}

	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	header, err := reader.Read()

	if err != nil {
		return DatasetInfo{}, fmt.Errorf(
			"no se pudo leer la cabecera: %w",
			err,
		)
	}

	rowCount := 0

	for {
		_, err := reader.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			return DatasetInfo{}, fmt.Errorf(
				"error leyendo el registro %d: %w",
				rowCount+1,
				err,
			)
		}

		rowCount++
	}

	return DatasetInfo{
		RowCount: rowCount,
		Columns:  header,
	}, nil
}
