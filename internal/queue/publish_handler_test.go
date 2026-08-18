package queue

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

type publishTestRuntime struct {
	published bool
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
