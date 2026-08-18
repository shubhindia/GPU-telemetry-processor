package processor

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func httpResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestNewHTTPClientValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		client  *http.Client
		baseURL string
		wantErr string
	}{
		{
			name:    "missing client",
			baseURL: "http://queue:8080",
			wantErr: "http client is required",
		},
		{
			name:    "invalid url",
			client:  http.DefaultClient,
			baseURL: "://bad",
			wantErr: "parse base url:",
		},
		{
			name:    "missing scheme",
			client:  http.DefaultClient,
			baseURL: "queue:8080",
			wantErr: "base url must include scheme and host",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client, err := NewHTTPClient(tc.client, tc.baseURL)
			if client != nil {
				t.Fatal("expected nil client")
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("NewHTTPClient() error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestHTTPClientConsumeSuccess(t *testing.T) {
	t.Parallel()

	client, err := NewHTTPClient(newTestClient(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/consume" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("topic"); got != "gpu" {
			t.Fatalf("topic = %q", got)
		}
		if got := r.URL.Query().Get("group"); got != "processor" {
			t.Fatalf("group = %q", got)
		}

		return httpResponse(http.StatusOK, `{"topic":"gpu","id":"message-1","routing_key":"GPU-1","payload":{"metric_name":"gpu_util","value":"42"}}`), nil
	}), "http://queue:8080")
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}

	message, ok, err := client.Consume(context.Background(), "gpu", "processor")
	if err != nil {
		t.Fatalf("Consume() error = %v", err)
	}
	if !ok {
		t.Fatal("expected message to be available")
	}
	if message.Topic != "gpu" || message.ID != "message-1" || message.RoutingKey != "GPU-1" {
		t.Fatalf("unexpected message metadata: %+v", message)
	}
	if string(message.Payload) != `{"metric_name":"gpu_util","value":"42"}` {
		t.Fatalf("unexpected payload %q", message.Payload)
	}
}

func TestHTTPClientConsumeNoContent(t *testing.T) {
	t.Parallel()

	client, err := NewHTTPClient(newTestClient(func(_ *http.Request) (*http.Response, error) {
		return httpResponse(http.StatusNoContent, ""), nil
	}), "http://queue:8080")
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}

	message, ok, err := client.Consume(context.Background(), "gpu", "processor")
	if err != nil {
		t.Fatalf("Consume() error = %v", err)
	}
	if ok {
		t.Fatal("expected no message")
	}
	if message.ID != "" || message.Topic != "" || message.RoutingKey != "" || len(message.Payload) != 0 {
		t.Fatalf("expected empty message, got %+v", message)
	}
}

func TestHTTPClientConsumeUnexpectedStatus(t *testing.T) {
	t.Parallel()

	client, err := NewHTTPClient(newTestClient(func(_ *http.Request) (*http.Response, error) {
		return httpResponse(http.StatusInternalServerError, "queue unhappy\n"), nil
	}), "http://queue:8080")
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}

	_, _, err = client.Consume(context.Background(), "gpu", "processor")
	if err == nil || !strings.Contains(err.Error(), "consume request failed with status 500: queue unhappy") {
		t.Fatalf("Consume() error = %v", err)
	}
}

func TestHTTPClientConsumeDecodeError(t *testing.T) {
	t.Parallel()

	client, err := NewHTTPClient(newTestClient(func(_ *http.Request) (*http.Response, error) {
		return httpResponse(http.StatusOK, `{"topic":`), nil
	}), "http://queue:8080")
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}

	_, _, err = client.Consume(context.Background(), "gpu", "processor")
	if err == nil || !strings.Contains(err.Error(), "decode consume response") {
		t.Fatalf("Consume() error = %v", err)
	}
}

func TestHTTPClientAck(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		client, err := NewHTTPClient(newTestClient(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/ack" {
				t.Fatalf("unexpected path %q", r.URL.Path)
			}
			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("content-type = %q", got)
			}

			var payload map[string]string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if payload["message_id"] != "message-1" {
				t.Fatalf("message_id = %q", payload["message_id"])
			}

			return httpResponse(http.StatusNoContent, ""), nil
		}), "http://queue:8080")
		if err != nil {
			t.Fatalf("NewHTTPClient() error = %v", err)
		}

		if err := client.Ack(context.Background(), "message-1"); err != nil {
			t.Fatalf("Ack() error = %v", err)
		}
	})

	t.Run("unexpected status", func(t *testing.T) {
		client, err := NewHTTPClient(newTestClient(func(_ *http.Request) (*http.Response, error) {
			return httpResponse(http.StatusConflict, "missing inflight message\n"), nil
		}), "http://queue:8080")
		if err != nil {
			t.Fatalf("NewHTTPClient() error = %v", err)
		}

		err = client.Ack(context.Background(), "message-1")
		if err == nil || !strings.Contains(err.Error(), "ack request failed with status 409: missing inflight message") {
			t.Fatalf("Ack() error = %v", err)
		}
	})
}

func TestUnexpectedStatusErrorReadFailure(t *testing.T) {
	t.Parallel()

	err := unexpectedStatusError("consume request", &http.Response{
		StatusCode: http.StatusBadGateway,
		Body:       io.NopCloser(failingReader{}),
	})
	if err == nil || !strings.Contains(err.Error(), "read body") {
		t.Fatalf("unexpectedStatusError() = %v", err)
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func (failingReader) Close() error {
	return nil
}

func TestHTTPResponseHelperStatusText(t *testing.T) {
	t.Parallel()

	resp := httpResponse(http.StatusCreated, "ok")
	if resp.Status != http.StatusText(http.StatusCreated) {
		t.Fatalf("status = %q", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !bytes.Equal(body, []byte("ok")) {
		t.Fatalf("body = %q", body)
	}
}
