package api

import (
	_ "embed"
	"net/http"
	"strings"
)

//go:generate go run github.com/swaggo/swag/cmd/swag@v1.16.4 init -g ../../cmd/api/main.go -o ./docs --outputTypes json --generatedTime=false --parseInternal

//go:embed docs/swagger.json
var swaggerSpec string

func SwaggerSpec() string {
	return swaggerSpec
}

func NewOpenAPIHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(swaggerSpec))
	})
}

func NewSwaggerUIHandler(specPath string) http.Handler {
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
        url: '__SPEC_PATH__',
        dom_id: '#swagger-ui',
        deepLinking: true,
        displayRequestDuration: true,
        tryItOutEnabled: true
      });
    };
  </script>
</body>
</html>`

	body := strings.ReplaceAll(page, "__SPEC_PATH__", specPath)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(body))
	})
}
