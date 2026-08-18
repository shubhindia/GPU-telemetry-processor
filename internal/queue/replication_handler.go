package queue

import (
	"encoding/json"
	"errors"
	"net/http"
)

type ReplicationHandler struct {
	storage Storage
}

func NewReplicationHandler(storage Storage) *ReplicationHandler {
	return &ReplicationHandler{
		storage: storage,
	}
}

func (h *ReplicationHandler) ServeHTTP(
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

	var request replicationRequest

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		http.Error(
			w,
			"invalid request",
			http.StatusBadRequest,
		)
		return
	}

	if err := h.storage.AppendRecord(
		r.Context(),
		request.PartitionID,
		request.Record,
	); err != nil {
		status := http.StatusInternalServerError

		if errors.Is(err, ErrUnexpectedOffset) {
			status = http.StatusConflict
		}

		http.Error(w, err.Error(), status)
		return
	}

	w.WriteHeader(http.StatusOK)
}
