package telemetry

import (
	"strings"
	"testing"
	"time"
)

func TestBuildQueryIncludesFiltersAndLimit(t *testing.T) {
	statement, args := buildQuery(Query{
		Start:      mustTime(t, "2026-08-18T08:00:00Z"),
		End:        mustTime(t, "2026-08-18T08:05:00Z"),
		MetricName: "gpu_util",
		UUID:       "GPU-1",
		Hostname:   "node-a",
		GPUID:      "0",
		Device:     "nvidia0",
		Limit:      10,
	})

	for _, want := range []string{
		"ts.recorded_at >= $1",
		"ts.recorded_at <= $2",
		"sr.metric_name = $3",
		"sr.uuid = $4",
		"sr.hostname = $5",
		"sr.gpu_id = $6",
		"sr.device = $7",
		"LIMIT $8",
	} {
		if !strings.Contains(statement, want) {
			t.Fatalf("expected query to contain %q, got:\n%s", want, statement)
		}
	}

	if len(args) != 8 {
		t.Fatalf("expected 8 args, got %d", len(args))
	}
	if args[2] != "gpu_util" || args[7] != 10 {
		t.Fatalf("unexpected args: %+v", args)
	}
}

func TestBuildQueryOmitsOptionalFilters(t *testing.T) {
	statement, args := buildQuery(Query{
		Start: mustTime(t, "2026-08-18T08:00:00Z"),
		End:   mustTime(t, "2026-08-18T08:05:00Z"),
	})

	if strings.Contains(statement, "sr.metric_name =") {
		t.Fatalf("did not expect optional filters, got:\n%s", statement)
	}
	if strings.Contains(statement, "LIMIT") {
		t.Fatalf("did not expect limit clause, got:\n%s", statement)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()

	timestamp, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("time.Parse() error = %v", err)
	}

	return timestamp
}
