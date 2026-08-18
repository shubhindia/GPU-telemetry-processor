package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckedInOpenAPISpecMatchesRuntimeSpec(t *testing.T) {
	path := filepath.Join("..", "..", "docs", "openapi.json")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}

	var checkedIn map[string]any
	if err := json.Unmarshal(contents, &checkedIn); err != nil {
		t.Fatalf("json.Unmarshal(checked-in spec) error = %v", err)
	}

	runtimeSpec := OpenAPISpec()

	checkedInJSON, err := json.Marshal(checkedIn)
	if err != nil {
		t.Fatalf("json.Marshal(checked-in spec) error = %v", err)
	}

	runtimeJSON, err := json.Marshal(runtimeSpec)
	if err != nil {
		t.Fatalf("json.Marshal(runtime spec) error = %v", err)
	}

	if string(checkedInJSON) != string(runtimeJSON) {
		t.Fatalf("checked-in OpenAPI spec is out of date; run `make swagger`")
	}
}
