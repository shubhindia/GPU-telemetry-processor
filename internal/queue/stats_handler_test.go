package queue

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatsHandlerRejectsUnsupportedMethod(t *testing.T) {
	handler := NewStatsHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/stats", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, recorder.Code)
	}
}

func TestStatsHandlerReturnsRuntimeAndReplicationStats(t *testing.T) {
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
				0: {Topic: "gpu", ID: "message-1", Payload: []byte(`{"value":95}`)},
			},
		}},
		HashPartitionRouter{},
		nil,
	)

	if _, ok, err := runtime.Poll(context.Background(), "gpu", "processor"); err != nil {
		t.Fatalf("Poll() error = %v", err)
	} else if !ok {
		t.Fatal("expected delivered message")
	}

	replication := &ReplicationMetrics{}
	replication.Attempts.Add(2)
	replication.Successes.Add(1)

	handler := NewStatsHandler(runtime, replication)
	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		Counters    QueueMetricsSnapshot       `json:"counters"`
		Partitions  []RuntimePartitionStats    `json:"partitions"`
		Consumers   []RuntimeConsumerStats     `json:"consumers"`
		Replication ReplicationMetricsSnapshot `json:"replication"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if response.Counters.Delivered != 1 || response.Counters.Inflight != 1 {
		t.Fatalf("unexpected counters: %+v", response.Counters)
	}
	if len(response.Partitions) != 1 || response.Partitions[0].Role != string(ReplicaLeader) {
		t.Fatalf("unexpected partitions: %+v", response.Partitions)
	}
	if len(response.Consumers) != 1 || response.Consumers[0].Group != "processor" {
		t.Fatalf("unexpected consumers: %+v", response.Consumers)
	}
	if response.Replication.Attempts != 2 || response.Replication.Successes != 1 {
		t.Fatalf("unexpected replication stats: %+v", response.Replication)
	}
}
