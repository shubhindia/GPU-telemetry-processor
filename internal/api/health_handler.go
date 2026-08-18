package api

import "net/http"

// HealthHandler godoc
//
//	@Summary	Health check
//	@Tags		System
//	@Produce	json
//	@Success	200	{object}	HealthResponse
//	@Router		/health [get]
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
