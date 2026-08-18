package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type consumeWrappedOffsetStore struct{}

func (consumeWrappedOffsetStore) Append(_ context.Context, _ int, _ Message) (Offset, error) {
	return 0, nil
}

func (consumeWrappedOffsetStore) AppendRecord(_ context.Context, _ int, _ Record) error {
	return nil
}

func (consumeWrappedOffsetStore) Read(_ context.Context, partitionID int, offset Offset) (Message, error) {
	return Message{}, fmt.Errorf("partition %d offset %d: %w", partitionID, offset, ErrOffsetNotFound)
}

func (consumeWrappedOffsetStore) Flush(_ context.Context, _ int) error {
	return nil
}

func TestConsumeHandlerRejectsUnsupportedMethod(t *testing.T) {
	handler := NewConsumeHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/consume", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected %d, got %d", http.StatusMethodNotAllowed, recorder.Code)
	}
}

func TestConsumeHandlerRequiresTopicAndGroup(t *testing.T) {
	handler := NewConsumeHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/consume", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/consume?topic=gpu", nil)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

func TestConsumeHandlerReturnsMessage(t *testing.T) {
	nodes := []Node{{ID: "queue-0", Address: "queue-0:9000"}}
	partitions := []Partition{{
		ID:       0,
		Replicas: []Replica{{NodeID: "queue-0", Role: ReplicaLeader}},
	}}

	runtime := NewRuntime(
		&runtimeTestCluster{nodes: nodes},
		*NewPartitionManager(nodes[0], partitions),
		&runtimeTestStorage{messages: map[int]map[Offset]Message{
			0: {
				0: {Topic: "gpu", ID: "message-1", RoutingKey: "GPU-123", Payload: []byte(`{"value":95}`)},
			},
		}},
		HashPartitionRouter{},
		nil,
	)

	handler := NewConsumeHandler(runtime)
	req := httptest.NewRequest(http.MethodGet, "/consume?topic=gpu&group=processor", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		Topic      string          `json:"topic"`
		ID         string          `json:"id"`
		RoutingKey string          `json:"routing_key"`
		Payload    json.RawMessage `json:"payload"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Topic != "gpu" || response.ID != "message-1" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestConsumeHandlerReturnsNoContentWhenEmpty(t *testing.T) {
	runtime := NewRuntime(
		&runtimeTestCluster{},
		PartitionManager{},
		&runtimeTestStorage{},
		HashPartitionRouter{},
		nil,
	)

	handler := NewConsumeHandler(runtime)
	req := httptest.NewRequest(http.MethodGet, "/consume?topic=gpu&group=processor", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, recorder.Code)
	}
}

func TestConsumeHandlerReturnsNoContentForWrappedOffsetNotFound(t *testing.T) {
	nodes := []Node{{ID: "queue-0", Address: "queue-0:9000"}}
	partitions := []Partition{{
		ID:       0,
		Replicas: []Replica{{NodeID: "queue-0", Role: ReplicaLeader}},
	}}

	runtime := NewRuntime(
		&runtimeTestCluster{nodes: nodes},
		*NewPartitionManager(nodes[0], partitions),
		consumeWrappedOffsetStore{},
		HashPartitionRouter{},
		nil,
	)

	handler := NewConsumeHandler(runtime)
	req := httptest.NewRequest(http.MethodGet, "/consume?topic=gpu&group=processor", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, recorder.Code)
	}
}
