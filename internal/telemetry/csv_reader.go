package telemetry

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
)

var requiredColumns = []string{
	"timestamp",
	"metric_name",
	"gpu_id",
	"device",
	"uuid",
	"modelName",
	"Hostname",
	"container",
	"pod",
	"namespace",
	"value",
	"labels_raw",
}

func LoadFile(path string) ([]Record, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open telemetry csv: %w", err)
	}
	defer file.Close()

	return ReadCSV(file)
}

func ReadCSV(reader io.Reader) ([]Record, error) {
	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1

	header, err := csvReader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("telemetry csv is empty")
		}

		return nil, fmt.Errorf("read telemetry header: %w", err)
	}

	columns := make(map[string]int, len(header))
	for index, name := range header {
		columns[name] = index
	}

	for _, name := range requiredColumns {
		if _, ok := columns[name]; !ok {
			return nil, fmt.Errorf("telemetry csv missing column %q", name)
		}
	}

	var records []Record

	for line := 2; ; line++ {
		row, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read telemetry row %d: %w", line, err)
		}

		records = append(records, Record{
			Timestamp:  field(row, columns, "timestamp"),
			MetricName: field(row, columns, "metric_name"),
			GPUID:      field(row, columns, "gpu_id"),
			Device:     field(row, columns, "device"),
			UUID:       field(row, columns, "uuid"),
			ModelName:  field(row, columns, "modelName"),
			Hostname:   field(row, columns, "Hostname"),
			Container:  field(row, columns, "container"),
			Pod:        field(row, columns, "pod"),
			Namespace:  field(row, columns, "namespace"),
			Value:      field(row, columns, "value"),
			LabelsRaw:  field(row, columns, "labels_raw"),
		})
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("telemetry csv has no rows")
	}

	return records, nil
}

func field(row []string, columns map[string]int, name string) string {
	index, ok := columns[name]
	if !ok || index >= len(row) {
		return ""
	}

	return row[index]
}
