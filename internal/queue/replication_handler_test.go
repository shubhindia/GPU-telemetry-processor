package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestReplicationHandlerAppendsRecord(t *testing.T) {
	store := NewPartitionStore(
		filepath.Join(t.TempDir(), "follower"),
		QueueConfig{},
	)

	if err := store.OpenPartition(0); err != nil {
		t.Fatalf("open partition: %v", err)
	}
	defer store.Close()

	handler := NewReplicationHandler(store)

	request := replicationRequest{
		PartitionID: 0,
		Record: Record{
			Offset: 0,
			Message: Message{
				ID:         "message-1",
				RoutingKey: "GPU-123",
				Payload:    []byte(`{"metric":"gpu_util","value":95}`),
			},
		},
	}

	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/internal/replicate",
		bytes.NewReader(body),
	)
	req = req.WithContext(context.Background())

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	message, err := store.Read(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("read replicated record: %v", err)
	}

	if message.ID != request.Record.Message.ID {
		t.Fatalf(
			"expected ID %q, got %q",
			request.Record.Message.ID,
			message.ID,
		)
	}

	if message.RoutingKey != request.Record.Message.RoutingKey {
		t.Fatalf(
			"expected routing key %q, got %q",
			request.Record.Message.RoutingKey,
			message.RoutingKey,
		)
	}

	if string(message.Payload) != string(request.Record.Message.Payload) {
		t.Fatalf(
			"expected payload %q, got %q",
			request.Record.Message.Payload,
			message.Payload,
		)
	}
}

func TestReplicationHandlerRejectsInvalidRequest(t *testing.T) {
	store := NewPartitionStore(
		filepath.Join(t.TempDir(), "follower"),
		QueueConfig{},
	)

	if err := store.OpenPartition(0); err != nil {
		t.Fatalf("open partition: %v", err)
	}
	defer store.Close()

	handler := NewReplicationHandler(store)

	req := httptest.NewRequest(
		http.MethodPost,
		"/internal/replicate",
		bytes.NewBufferString("{invalid"),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestReplicationHandlerRejectsUnsupportedMethod(t *testing.T) {
	store := NewPartitionStore(
		filepath.Join(t.TempDir(), "follower"),
		QueueConfig{},
	)

	handler := NewReplicationHandler(store)

	req := httptest.NewRequest(
		http.MethodGet,
		"/internal/replicate",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusMethodNotAllowed,
			recorder.Code,
		)
	}
}

func TestReplicationHandlerReturnsConflictForUnexpectedOffset(t *testing.T) {
	store := NewPartitionStore(
		filepath.Join(t.TempDir(), "follower"),
		QueueConfig{},
	)

	if err := store.OpenPartition(0); err != nil {
		t.Fatalf("open partition: %v", err)
	}
	defer store.Close()

	handler := NewReplicationHandler(store)

	request := replicationRequest{
		PartitionID: 0,
		Record: Record{
			Offset: 42,
			Message: Message{
				ID: "message-42",
			},
		},
	}

	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/internal/replicate",
		bytes.NewReader(body),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusConflict {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusConflict,
			recorder.Code,
		)
	}
}
