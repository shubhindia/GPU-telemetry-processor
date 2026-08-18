package queue

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestPublishHandlerRejectsUnsupportedMethod(t *testing.T) {
	handler := NewPublishHandler(nil)

	req := httptest.NewRequest(
		http.MethodGet,
		"/publish",
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

func TestPublishHandlerRejectsInvalidJSON(t *testing.T) {
	handler := NewPublishHandler(nil)

	req := httptest.NewRequest(
		http.MethodPost,
		"/publish",
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

func TestPublishHandlerRejectsMissingTopic(t *testing.T) {
	handler := NewPublishHandler(nil)

	req := httptest.NewRequest(
		http.MethodPost,
		"/publish",
		bytes.NewBufferString(`{
			"id": "message-1"
		}`),
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

func TestPublishHandlerRejectsMissingID(t *testing.T) {
	handler := NewPublishHandler(nil)

	req := httptest.NewRequest(
		http.MethodPost,
		"/publish",
		bytes.NewBufferString(`{
			"topic": "gpu"
		}`),
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

func TestPublishHandlerForwardsToPartitionLeader(t *testing.T) {
	leaderReceived := false

	nodes := []Node{
		{ID: "queue-0", Address: "http://queue-0.default.svc:8080"},
		{ID: "queue-1", Address: "http://queue-1.default.svc:8080"},
	}
	partitions := []Partition{{
		ID: 0,
		Replicas: []Replica{
			{NodeID: "queue-0", Role: ReplicaLeader},
			{NodeID: "queue-1", Role: ReplicaFollower},
		},
	}}

	runtime := NewRuntime(
		&runtimeTestCluster{nodes: nodes},
		*NewPartitionManager(nodes[1], partitions),
		&runtimeTestStorage{},
		HashPartitionRouter{},
		nil,
	)

	handler := NewPublishHandler(runtime)
	handler.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		leaderReceived = true

		if r.URL.String() != "http://queue-0.default.svc:8080/publish" {
			t.Fatalf("expected forwarded url %q, got %q", "http://queue-0.default.svc:8080/publish", r.URL.String())
		}

		var request publishRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode forwarded request: %v", err)
		}

		if request.Topic != "gpu" || request.ID != "message-1" || request.RoutingKey != "GPU-123" {
			t.Fatalf("unexpected forwarded request: %+v", request)
		}

		return &http.Response{
			StatusCode: http.StatusAccepted,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"status":"ok"}`)),
		}, nil
	})}

	req := httptest.NewRequest(
		http.MethodPost,
		"/publish",
		bytes.NewBufferString(`{
			"topic": "gpu",
			"id": "message-1",
			"routing_key": "GPU-123",
			"payload": {"value": 95}
		}`),
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, recorder.Code)
	}
	if !leaderReceived {
		t.Fatal("expected request to be forwarded to leader")
	}
	if body := recorder.Body.String(); body != `{"status":"ok"}` {
		t.Fatalf("unexpected response body %q", body)
	}
}
