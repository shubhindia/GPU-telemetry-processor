package queue

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsHandlerRejectsUnsupportedMethod(t *testing.T) {
	handler := NewMetricsHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/metrics", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, recorder.Code)
	}
}

func TestMetricsHandlerExposesPrometheusMetrics(t *testing.T) {
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
	replication.Attempts.Add(3)
	replication.Failures.Add(1)

	handler := NewMetricsHandler(runtime, replication)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	body := recorder.Body.String()
	for _, want := range []string{
		"# TYPE queue_messages_delivered_total counter",
		"queue_messages_delivered_total 1",
		"queue_inflight_messages 1",
		`queue_partition_next_offset{partition="0",role="leader"} 1`,
		`queue_consumer_next_offset{topic="gpu",group="processor",partition="0"} 0`,
		`queue_consumer_inflight{topic="gpu",group="processor",partition="0"} 1`,
		"queue_replication_attempts_total 3",
		"queue_replication_failures_total 1",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected metrics body to contain %q, got:\n%s", want, body)
		}
	}
}
