package queue

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type replicationRoundTripFunc func(*http.Request) (*http.Response, error)

func (f replicationRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newReplicationTestClient(fn replicationRoundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func newReplicationResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestHTTPReplicationTransport(t *testing.T) {
	transport := NewHTTPReplicationTransport(
		newReplicationTestClient(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}

			if r.URL.String() != "http://queue-1:8080/internal/replicate" {
				t.Fatalf("expected target url, got %s", r.URL.String())
			}
			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("content-type = %q", got)
			}

			var request replicationRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode request: %v", err)
			}

			if request.PartitionID != 7 {
				t.Fatalf("expected partition ID 7, got %d", request.PartitionID)
			}
			if request.Record.Offset != 42 {
				t.Fatalf("expected offset 42, got %d", request.Record.Offset)
			}
			if request.Record.Message.ID != "message-42" {
				t.Fatalf("expected message ID message-42, got %q", request.Record.Message.ID)
			}

			return newReplicationResponse(http.StatusOK, ""), nil
		}),
		"/internal/replicate",
	)

	node := Node{
		ID:      "queue-1",
		Address: "http://queue-1:8080",
	}

	record := Record{
		Offset: 42,
		Message: Message{
			ID:         "message-42",
			RoutingKey: "GPU-123",
			Payload:    []byte("payload"),
		},
	}

	err := transport.Replicate(context.Background(), node, 7, record)
	if err != nil {
		t.Fatalf("replication failed: %v", err)
	}
}

func TestHTTPReplicationTransportReturnsErrorForNonSuccess(t *testing.T) {
	transport := NewHTTPReplicationTransport(
		newReplicationTestClient(func(_ *http.Request) (*http.Response, error) {
			return newReplicationResponse(http.StatusServiceUnavailable, "replication failed\n"), nil
		}),
		"/internal/replicate",
	)

	node := Node{
		ID:      "queue-1",
		Address: "http://queue-1:8080",
	}

	err := transport.Replicate(context.Background(), node, 7, Record{Offset: 42})
	if err == nil || !strings.Contains(err.Error(), "replication request to http://queue-1:8080/internal/replicate failed") {
		t.Fatalf("Replicate() error = %v", err)
	}
}
