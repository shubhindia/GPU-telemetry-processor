package api

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/shubhindia/gpu-telemetry/internal/telemetry"
)

type gpuListResponse struct {
	Items []telemetry.GPU `json:"items"`
}

type gpuTelemetryResponse struct {
	GPUId string                   `json:"gpu_id"`
	Items []telemetry.SampleRecord `json:"items"`
}

type telemetryQueryResponse struct {
	Items []telemetry.SampleRecord `json:"items"`
}

func NewOpenAPIHandler() http.Handler {
	spec := OpenAPISpec()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(spec)
	})
}

func NewSwaggerUIHandler() http.Handler {
	const page = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>GPU Telemetry API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function() {
      window.ui = SwaggerUIBundle({
        url: '/openapi.json',
        dom_id: '#swagger-ui',
        deepLinking: true,
        displayRequestDuration: true,
        tryItOutEnabled: true
      });
    };
  </script>
</body>
</html>`

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(page))
	})
}

func OpenAPISpec() map[string]any {
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "GPU Telemetry API",
			"version":     "1.0.0",
			"description": "Query processed GPU telemetry samples by time window and optional filters.",
		},
		"paths": map[string]any{
			"/health": map[string]any{
				"get": operationSpec("Health check", "getHealth", nil, map[string]any{
					"200": map[string]any{"description": "API is healthy"},
				}),
			},
			"/api/v1/gpus": map[string]any{
				"get": operationSpec("List GPUs", "listGPUs", nil, map[string]any{
					"200": jsonResponse("Available GPUs", schemaRef("GPUListResponse")),
					"500": map[string]any{"description": "Internal server error"},
				}),
			},
			"/api/v1/gpus/{id}/telemetry": map[string]any{
				"get": operationSpec("Query telemetry for a GPU", "listGPUTelemetry", []map[string]any{
					pathParameter("id", "GPU UUID."),
					dateTimeParameter("start_time", true, "Start of the query window in RFC3339 format."),
					dateTimeParameter("end_time", true, "End of the query window in RFC3339 format."),
					stringParameter("metric_name", false, "query", "Optional metric name filter."),
					stringParameter("hostname", false, "query", "Optional hostname filter."),
					stringParameter("gpu_id", false, "query", "Optional GPU ID filter."),
					stringParameter("device", false, "query", "Optional device filter."),
					integerParameter("limit", false, "query", "Optional result limit."),
				}, map[string]any{
					"200": jsonResponse("Telemetry query result", schemaRef("GPUTelemetryResponse")),
					"400": map[string]any{"description": "Invalid query parameters"},
					"500": map[string]any{"description": "Internal server error"},
				}),
			},
			"/telemetry": map[string]any{
				"get": deprecatedOperationSpec("Query telemetry samples", "listTelemetry", []map[string]any{
					dateTimeParameter("start", true, "Start of the query window in RFC3339 format."),
					dateTimeParameter("end", true, "End of the query window in RFC3339 format."),
					stringParameter("metric_name", false, "query", "Optional metric name filter."),
					stringParameter("uuid", false, "query", "Optional GPU UUID filter."),
					stringParameter("hostname", false, "query", "Optional hostname filter."),
					stringParameter("gpu_id", false, "query", "Optional GPU ID filter."),
					stringParameter("device", false, "query", "Optional device filter."),
					integerParameter("limit", false, "query", "Optional result limit."),
				}, map[string]any{
					"200": jsonResponse("Telemetry query result", schemaRef("TelemetryQueryResponse")),
					"400": map[string]any{"description": "Invalid query parameters"},
					"500": map[string]any{"description": "Internal server error"},
				}),
			},
		},
		"components": map[string]any{
			"schemas": map[string]any{
				"GPU":                    schemaFromType(reflect.TypeFor[telemetry.GPU]()),
				"TelemetrySample":        schemaFromType(reflect.TypeFor[telemetry.SampleRecord]()),
				"GPUListResponse":        schemaFromType(reflect.TypeFor[gpuListResponse]()),
				"GPUTelemetryResponse":   schemaFromType(reflect.TypeFor[gpuTelemetryResponse]()),
				"TelemetryQueryResponse": schemaFromType(reflect.TypeFor[telemetryQueryResponse]()),
			},
		},
	}
}

func operationSpec(summary string, operationID string, parameters []map[string]any, responses map[string]any) map[string]any {
	operation := map[string]any{
		"summary":     summary,
		"operationId": operationID,
		"responses":   responses,
	}

	if len(parameters) > 0 {
		operation["parameters"] = parameters
	}

	return operation
}

func deprecatedOperationSpec(summary string, operationID string, parameters []map[string]any, responses map[string]any) map[string]any {
	operation := operationSpec(summary, operationID, parameters, responses)
	operation["deprecated"] = true
	return operation
}

func jsonResponse(description string, schema map[string]any) map[string]any {
	return map[string]any{
		"description": description,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": schema,
			},
		},
	}
}

func schemaRef(name string) map[string]any {
	return map[string]any{"$ref": "#/components/schemas/" + name}
}

func pathParameter(name string, description string) map[string]any {
	return stringParameter(name, true, "path", description)
}

func dateTimeParameter(name string, required bool, description string) map[string]any {
	return map[string]any{
		"name":        name,
		"in":          "query",
		"required":    required,
		"description": description,
		"schema": map[string]any{
			"type":   "string",
			"format": "date-time",
		},
	}
}

func stringParameter(name string, required bool, location string, description string) map[string]any {
	return map[string]any{
		"name":        name,
		"in":          location,
		"required":    required,
		"description": description,
		"schema": map[string]any{
			"type": "string",
		},
	}
}

func integerParameter(name string, required bool, location string, description string) map[string]any {
	return map[string]any{
		"name":        name,
		"in":          location,
		"required":    required,
		"description": description,
		"schema": map[string]any{
			"type":    "integer",
			"minimum": 0,
		},
	}
}

func schemaFromType(typ reflect.Type) map[string]any {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	if typ == reflect.TypeFor[time.Time]() {
		return map[string]any{
			"type":   "string",
			"format": "date-time",
		}
	}

	switch typ.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Slice, reflect.Array:
		return map[string]any{
			"type":  "array",
			"items": schemaFromType(typ.Elem()),
		}
	case reflect.Struct:
		properties := make(map[string]any)
		for index := range typ.NumField() {
			field := typ.Field(index)
			if !field.IsExported() {
				continue
			}

			name, ok := jsonFieldName(field)
			if !ok {
				continue
			}

			properties[name] = schemaFromType(field.Type)
		}

		return map[string]any{
			"type":       "object",
			"properties": properties,
		}
	default:
		return map[string]any{"type": "string"}
	}
}

func jsonFieldName(field reflect.StructField) (string, bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", false
	}

	name := strings.Split(tag, ",")[0]
	if name == "" {
		name = field.Name
	}

	return name, true
}
