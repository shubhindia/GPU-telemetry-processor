package queue

import (
	"encoding/json"
	"net/http"
)

type PublishHandler struct {
	runtime *Runtime
}

func NewPublishHandler(runtime *Runtime) *PublishHandler {
	return &PublishHandler{
		runtime: runtime,
	}
}

type publishRequest struct {
	Topic      string          `json:"topic"`
	ID         string          `json:"id"`
	RoutingKey string          `json:"routing_key"`
	Payload    json.RawMessage `json:"payload"`
}

func (h *PublishHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	var request publishRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(
			w,
			"invalid request",
			http.StatusBadRequest,
		)
		return
	}

	if request.Topic == "" {
		http.Error(
			w,
			"topic is required",
			http.StatusBadRequest,
		)
		return
	}

	if request.ID == "" {
		http.Error(
			w,
			"id is required",
			http.StatusBadRequest,
		)
		return
	}

	if len(request.Payload) == 0 {
		http.Error(
			w,
			"payload is required",
			http.StatusBadRequest,
		)
		return
	}

	err := h.runtime.Publish(
		r.Context(),
		request.Topic,
		Message{
			ID:         request.ID,
			RoutingKey: request.RoutingKey,
			Payload:    []byte(request.Payload),
		},
	)
	if err != nil {
		status := http.StatusInternalServerError

		if err == ErrNotPartitionLeader {
			status = http.StatusConflict
		}

		if err == ErrReplicationQuorum {
			status = http.StatusServiceUnavailable
		}

		http.Error(w, err.Error(), status)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
