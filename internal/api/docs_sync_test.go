package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckedInSwaggerSpecMatchesGeneratedDoc(t *testing.T) {
	path := filepath.Join("docs", "swagger.json")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}

	var checkedIn map[string]any
	if err := json.Unmarshal(contents, &checkedIn); err != nil {
		t.Fatalf("json.Unmarshal(checked-in spec) error = %v", err)
	}

	var generated map[string]any
	if err := json.Unmarshal([]byte(SwaggerSpec()), &generated); err != nil {
		t.Fatalf("json.Unmarshal(generated spec) error = %v", err)
	}

	checkedInJSON, err := json.Marshal(checkedIn)
	if err != nil {
		t.Fatalf("json.Marshal(checked-in spec) error = %v", err)
	}

	generatedJSON, err := json.Marshal(generated)
	if err != nil {
		t.Fatalf("json.Marshal(generated spec) error = %v", err)
	}

	if string(checkedInJSON) != string(generatedJSON) {
		t.Fatalf("checked-in Swagger spec is out of date; run `make swagger`")
	}
}
