package queue

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type PublishHandler struct {
	runtime *Runtime
	client  *http.Client
}

func NewPublishHandler(runtime *Runtime) *PublishHandler {
	return &PublishHandler{
		runtime: runtime,
		client:  http.DefaultClient,
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

	message := Message{
		Topic:      request.Topic,
		ID:         request.ID,
		RoutingKey: request.RoutingKey,
		Payload:    []byte(request.Payload),
	}

	leader, _, local, err := h.runtime.PartitionLeader(
		r.Context(),
		request.Topic,
		message,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !local {
		h.forwardToLeader(w, r, leader, request)
		return
	}

	err = h.runtime.Publish(
		r.Context(),
		request.Topic,
		message,
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

func (h *PublishHandler) forwardToLeader(
	w http.ResponseWriter,
	r *http.Request,
	leader Node,
	request publishRequest,
) {
	payload, err := json.Marshal(request)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	url := strings.TrimRight(leader.Address, "/") + "/publish"
	forwarded, err := http.NewRequestWithContext(
		r.Context(),
		http.MethodPost,
		url,
		bytes.NewReader(payload),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	forwarded.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(forwarded)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if contentType := resp.Header.Get("Content-Type"); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
