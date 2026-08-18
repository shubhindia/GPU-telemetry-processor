package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/shubhindia/gpu-telemetry/internal/queue"
)

type producerRoundTripFunc func(*http.Request) (*http.Response, error)

func (f producerRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newProducerTestClient(fn producerRoundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func producerResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestNewHTTPProducer(t *testing.T) {
	t.Parallel()

	producer, err := NewHTTPProducer(nil, "http://queue:8080/api")
	if err != nil {
		t.Fatalf("NewHTTPProducer() error = %v", err)
	}
	if producer.client == nil {
		t.Fatal("expected default client")
	}
	if producer.publishURL != "http://queue:8080/api/publish" {
		t.Fatalf("publishURL = %q", producer.publishURL)
	}
}

func TestNewHTTPProducerInvalidURL(t *testing.T) {
	t.Parallel()

	producer, err := NewHTTPProducer(http.DefaultClient, "://bad")
	if producer != nil {
		t.Fatal("expected nil producer")
	}
	if err == nil || !strings.Contains(err.Error(), "parse queue url") {
		t.Fatalf("NewHTTPProducer() error = %v", err)
	}
}

func TestHTTPProducerPublish(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		producer, err := NewHTTPProducer(newProducerTestClient(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/publish" {
				t.Fatalf("unexpected path %q", r.URL.Path)
			}
			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("content-type = %q", got)
			}

			var body struct {
				Topic      string          `json:"topic"`
				ID         string          `json:"id"`
				RoutingKey string          `json:"routing_key"`
				Payload    json.RawMessage `json:"payload"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body.Topic != "gpu" || body.ID != "message-1" || body.RoutingKey != "GPU-1" {
				t.Fatalf("unexpected request metadata: %+v", body)
			}
			if string(body.Payload) != `{"metric_name":"gpu_util"}` {
				t.Fatalf("unexpected payload %q", body.Payload)
			}

			return producerResponse(http.StatusAccepted, ""), nil
		}), "http://queue:8080")
		if err != nil {
			t.Fatalf("NewHTTPProducer() error = %v", err)
		}

		err = producer.Publish(context.Background(), "gpu", queue.Message{
			ID:         "message-1",
			RoutingKey: "GPU-1",
			Payload:    []byte(`{"metric_name":"gpu_util"}`),
		})
		if err != nil {
			t.Fatalf("Publish() error = %v", err)
		}
	})

	t.Run("unexpected status with body", func(t *testing.T) {
		producer, err := NewHTTPProducer(newProducerTestClient(func(_ *http.Request) (*http.Response, error) {
			return producerResponse(http.StatusServiceUnavailable, "replication quorum not reached\n"), nil
		}), "http://queue:8080")
		if err != nil {
			t.Fatalf("NewHTTPProducer() error = %v", err)
		}

		err = producer.Publish(context.Background(), "gpu", queue.Message{Payload: []byte(`{}`)})
		if err == nil || !strings.Contains(err.Error(), "publish message: 503 Service Unavailable: replication quorum not reached") {
			t.Fatalf("Publish() error = %v", err)
		}
	})

	t.Run("unexpected status without body", func(t *testing.T) {
		producer, err := NewHTTPProducer(newProducerTestClient(func(_ *http.Request) (*http.Response, error) {
			return producerResponse(http.StatusBadGateway, ""), nil
		}), "http://queue:8080")
		if err != nil {
			t.Fatalf("NewHTTPProducer() error = %v", err)
		}

		err = producer.Publish(context.Background(), "gpu", queue.Message{Payload: []byte(`{}`)})
		if err == nil || !strings.Contains(err.Error(), "publish message: 502 Bad Gateway: Bad Gateway") {
			t.Fatalf("Publish() error = %v", err)
		}
	})
}
