package api

import (
	"encoding/json"
	"net/http"
)

func NewOpenAPIHandler() http.Handler {
	spec := openAPISpec()

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

func openAPISpec() map[string]any {
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "GPU Telemetry API",
			"version":     "1.0.0",
			"description": "Query processed GPU telemetry samples by time window and optional filters.",
		},
		"paths": map[string]any{
			"/health": map[string]any{
				"get": map[string]any{
					"summary":     "Health check",
					"operationId": "getHealth",
					"responses": map[string]any{
						"200": map[string]any{
							"description": "API is healthy",
						},
					},
				},
			},
			"/api/v1/gpus": map[string]any{
				"get": map[string]any{
					"summary":     "List GPUs",
					"operationId": "listGPUs",
					"responses": map[string]any{
						"200": map[string]any{
							"description": "Available GPUs",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{
										"type": "object",
										"properties": map[string]any{
											"items": map[string]any{
												"type":  "array",
												"items": map[string]any{"$ref": "#/components/schemas/GPU"},
											},
										},
									},
								},
							},
						},
						"500": map[string]any{"description": "Internal server error"},
					},
				},
			},
			"/api/v1/gpus/{id}/telemetry": map[string]any{
				"get": map[string]any{
					"summary":     "Query telemetry for a GPU",
					"operationId": "listGPUTelemetry",
					"parameters": []map[string]any{
						{"name": "id", "in": "path", "required": true, "description": "GPU UUID.", "schema": map[string]any{"type": "string"}},
						{"name": "start_time", "in": "query", "required": true, "description": "Start of the query window in RFC3339 format.", "schema": map[string]any{"type": "string", "format": "date-time"}},
						{"name": "end_time", "in": "query", "required": true, "description": "End of the query window in RFC3339 format.", "schema": map[string]any{"type": "string", "format": "date-time"}},
						{"name": "metric_name", "in": "query", "required": false, "schema": map[string]any{"type": "string"}},
						{"name": "hostname", "in": "query", "required": false, "schema": map[string]any{"type": "string"}},
						{"name": "gpu_id", "in": "query", "required": false, "schema": map[string]any{"type": "string"}},
						{"name": "device", "in": "query", "required": false, "schema": map[string]any{"type": "string"}},
						{"name": "limit", "in": "query", "required": false, "schema": map[string]any{"type": "integer", "minimum": 0}},
					},
					"responses": map[string]any{
						"200": map[string]any{
							"description": "Telemetry query result",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{
										"type": "object",
										"properties": map[string]any{
											"gpu_id": map[string]any{
												"type": "string",
											},
											"items": map[string]any{
												"type":  "array",
												"items": map[string]any{"$ref": "#/components/schemas/TelemetrySample"},
											},
										},
									},
								},
							},
						},
						"400": map[string]any{"description": "Invalid query parameters"},
						"500": map[string]any{"description": "Internal server error"},
					},
				},
			},
			"/telemetry": map[string]any{
				"get": map[string]any{
					"summary":     "Query telemetry samples",
					"operationId": "listTelemetry",
					"deprecated":  true,
					"parameters": []map[string]any{
						{"name": "start", "in": "query", "required": true, "description": "Start of the query window in RFC3339 format.", "schema": map[string]any{"type": "string", "format": "date-time"}},
						{"name": "end", "in": "query", "required": true, "description": "End of the query window in RFC3339 format.", "schema": map[string]any{"type": "string", "format": "date-time"}},
						{"name": "metric_name", "in": "query", "required": false, "schema": map[string]any{"type": "string"}},
						{"name": "uuid", "in": "query", "required": false, "schema": map[string]any{"type": "string"}},
						{"name": "hostname", "in": "query", "required": false, "schema": map[string]any{"type": "string"}},
						{"name": "gpu_id", "in": "query", "required": false, "schema": map[string]any{"type": "string"}},
						{"name": "device", "in": "query", "required": false, "schema": map[string]any{"type": "string"}},
						{"name": "limit", "in": "query", "required": false, "schema": map[string]any{"type": "integer", "minimum": 0}},
					},
					"responses": map[string]any{
						"200": map[string]any{"description": "Telemetry query result"},
						"400": map[string]any{"description": "Invalid query parameters"},
						"500": map[string]any{"description": "Internal server error"},
					},
				},
			},
		},
		"components": map[string]any{
			"schemas": map[string]any{
				"GPU": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id":        map[string]any{"type": "string"},
						"gpu_id":    map[string]any{"type": "string"},
						"device":    map[string]any{"type": "string"},
						"uuid":      map[string]any{"type": "string"},
						"modelName": map[string]any{"type": "string"},
						"Hostname":  map[string]any{"type": "string"},
					},
				},
				"TelemetrySample": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"timestamp":   map[string]any{"type": "string", "format": "date-time"},
						"metric_name": map[string]any{"type": "string"},
						"gpu_id":      map[string]any{"type": "string"},
						"device":      map[string]any{"type": "string"},
						"uuid":        map[string]any{"type": "string"},
						"modelName":   map[string]any{"type": "string"},
						"Hostname":    map[string]any{"type": "string"},
						"container":   map[string]any{"type": "string"},
						"pod":         map[string]any{"type": "string"},
						"namespace":   map[string]any{"type": "string"},
						"value":       map[string]any{"type": "number"},
						"labels_raw":  map[string]any{"type": "string"},
					},
				},
			},
		},
	}
}
