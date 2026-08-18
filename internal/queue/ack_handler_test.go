package queue

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAckHandlerRejectsUnsupportedMethod(t *testing.T) {
	handler := NewAckHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/ack", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected %d, got %d", http.StatusMethodNotAllowed, recorder.Code)
	}
}

func TestAckHandlerRejectsInvalidRequest(t *testing.T) {
	handler := NewAckHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/ack", bytes.NewBufferString("{invalid"))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

func TestAckHandlerAcknowledgesMessage(t *testing.T) {
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

	if _, ok, err := runtime.Poll(context.Background(), "gpu", "processor"); err != nil || !ok {
		t.Fatalf("prepare poll failed, ok=%v err=%v", ok, err)
	}

	handler := NewAckHandler(runtime)
	req := httptest.NewRequest(http.MethodPost, "/ack", bytes.NewBufferString(`{"message_id":"message-1"}`))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, recorder.Code)
	}
}

func TestAckHandlerReturnsNotFoundForUnknownMessage(t *testing.T) {
	runtime := NewRuntime(
		&runtimeTestCluster{},
		PartitionManager{},
		&runtimeTestStorage{},
		HashPartitionRouter{},
		nil,
	)

	handler := NewAckHandler(runtime)
	req := httptest.NewRequest(http.MethodPost, "/ack", bytes.NewBufferString(`{"message_id":"missing"}`))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, recorder.Code)
	}
}
