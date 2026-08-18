package queue

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPReplicationTransport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/internal/replicate" {
			t.Fatalf("expected /internal/replicate, got %s", r.URL.Path)
		}

		var request replicationRequest

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		if request.PartitionID != 7 {
			t.Fatalf(
				"expected partition ID 7, got %d",
				request.PartitionID,
			)
		}

		if request.Record.Offset != 42 {
			t.Fatalf(
				"expected offset 42, got %d",
				request.Record.Offset,
			)
		}

		if request.Record.Message.ID != "message-42" {
			t.Fatalf(
				"expected message ID message-42, got %q",
				request.Record.Message.ID,
			)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := NewHTTPReplicationTransport(
		server.Client(),
		"/internal/replicate",
	)

	node := Node{
		ID:      "queue-1",
		Address: server.URL,
	}

	record := Record{
		Offset: 42,
		Message: Message{
			ID:         "message-42",
			RoutingKey: "GPU-123",
			Payload:    []byte("payload"),
		},
	}

	err := transport.Replicate(
		context.Background(),
		node,
		7,
		record,
	)
	if err != nil {
		t.Fatalf("replication failed: %v", err)
	}
}

func TestHTTPReplicationTransportReturnsErrorForNonSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		http.Error(
			w,
			"replication failed",
			http.StatusServiceUnavailable,
		)
	}))
	defer server.Close()

	transport := NewHTTPReplicationTransport(
		server.Client(),
		"/internal/replicate",
	)

	node := Node{
		ID:      "queue-1",
		Address: server.URL,
	}

	err := transport.Replicate(
		context.Background(),
		node,
		7,
		Record{Offset: 42},
	)
	if err == nil {
		t.Fatal("expected replication to fail")
	}
}
