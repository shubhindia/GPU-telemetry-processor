package telemetry

import (
	"strings"
	"testing"
)

func TestReadCSV(t *testing.T) {
	t.Parallel()

	records, err := ReadCSV(strings.NewReader(`timestamp,metric_name,gpu_id,device,uuid,modelName,Hostname,container,pod,namespace,value,labels_raw
2025-07-18T20:42:34Z,DCGM_FI_DEV_GPU_UTIL,0,nvidia0,GPU-123,NVIDIA H100 80GB HBM3,host-a,,,,95,"UUID=""GPU-123"",device=""nvidia0"""
`))
	if err != nil {
		t.Fatalf("ReadCSV() error = %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	record := records[0]
	if record.Timestamp != "2025-07-18T20:42:34Z" {
		t.Fatalf("expected original timestamp, got %q", record.Timestamp)
	}
	if record.MetricName != "DCGM_FI_DEV_GPU_UTIL" {
		t.Fatalf("expected metric name to be parsed, got %q", record.MetricName)
	}
	if record.UUID != "GPU-123" {
		t.Fatalf("expected UUID to be parsed, got %q", record.UUID)
	}
	if record.LabelsRaw == "" {
		t.Fatal("expected labels_raw to be preserved")
	}
}

func TestReadCSVMissingColumn(t *testing.T) {
	t.Parallel()

	_, err := ReadCSV(strings.NewReader("timestamp,metric_name\n2025-07-18T20:42:34Z,DCGM_FI_DEV_GPU_UTIL\n"))
	if err == nil {
		t.Fatal("expected missing column error")
	}
}
