package queue

import (
	"encoding/json"
	"net/http"
)

type ConsumeHandler struct {
	runtime *Runtime
}

func NewConsumeHandler(runtime *Runtime) *ConsumeHandler {
	return &ConsumeHandler{runtime: runtime}
}

func (h *ConsumeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	topic := r.URL.Query().Get("topic")
	if topic == "" {
		http.Error(w, "topic is required", http.StatusBadRequest)
		return
	}

	group := r.URL.Query().Get("group")
	if group == "" {
		http.Error(w, "group is required", http.StatusBadRequest)
		return
	}

	message, ok, err := h.runtime.Poll(r.Context(), topic, group)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct {
		Topic      string          `json:"topic"`
		ID         string          `json:"id"`
		RoutingKey string          `json:"routing_key"`
		Payload    json.RawMessage `json:"payload"`
	}{
		Topic:      message.Topic,
		ID:         message.ID,
		RoutingKey: message.RoutingKey,
		Payload:    json.RawMessage(message.Payload),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
