package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/shubhindia/gpu-telemetry/internal/queue"
)

type HTTPProducer struct {
	client     *http.Client
	publishURL string
}

func NewHTTPProducer(client *http.Client, queueURL string) (*HTTPProducer, error) {
	if client == nil {
		client = http.DefaultClient
	}

	parsedURL, err := url.Parse(queueURL)
	if err != nil {
		return nil, fmt.Errorf("parse queue url: %w", err)
	}

	parsedURL.Path = strings.TrimRight(parsedURL.Path, "/") + "/publish"

	return &HTTPProducer{
		client:     client,
		publishURL: parsedURL.String(),
	}, nil
}

func (p *HTTPProducer) Publish(
	ctx context.Context,
	topic string,
	message queue.Message,
) error {
	requestBody, err := json.Marshal(struct {
		Topic      string          `json:"topic"`
		ID         string          `json:"id"`
		RoutingKey string          `json:"routing_key"`
		Payload    json.RawMessage `json:"payload"`
	}{
		Topic:      topic,
		ID:         message.ID,
		RoutingKey: message.RoutingKey,
		Payload:    json.RawMessage(message.Payload),
	})
	if err != nil {
		return fmt.Errorf("marshal publish request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.publishURL,
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return fmt.Errorf("create publish request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("publish message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusAccepted {
		return nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	messageText := strings.TrimSpace(string(body))
	if messageText == "" {
		messageText = http.StatusText(resp.StatusCode)
	}

	return fmt.Errorf("publish message: %s: %s", resp.Status, messageText)
}
