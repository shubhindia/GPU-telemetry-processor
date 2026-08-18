package queue

import (
	"encoding/json"
	"net/http"
)

type AckHandler struct {
	runtime *Runtime
}

func NewAckHandler(runtime *Runtime) *AckHandler {
	return &AckHandler{runtime: runtime}
}

func (h *AckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		MessageID string `json:"message_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if request.MessageID == "" {
		http.Error(w, "message_id is required", http.StatusBadRequest)
		return
	}

	err := h.runtime.Ack(r.Context(), request.MessageID)
	if err != nil {
		status := http.StatusInternalServerError
		if err == ErrMessageNotInflight {
			status = http.StatusNotFound
		}

		http.Error(w, err.Error(), status)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
