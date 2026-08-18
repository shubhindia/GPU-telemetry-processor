package processor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/shubhindia/gpu-telemetry/internal/queue"
)

type Consumer interface {
	Consume(ctx context.Context, topic string, group string) (queue.Message, bool, error)
	Ack(ctx context.Context, messageID string) error
}

type HTTPClient struct {
	client  *http.Client
	baseURL *url.URL
}

func NewHTTPClient(client *http.Client, baseURL string) (*HTTPClient, error) {
	if client == nil {
		return nil, fmt.Errorf("http client is required")
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("base url must include scheme and host")
	}

	return &HTTPClient{client: client, baseURL: parsed}, nil
}

func (c *HTTPClient) Consume(
	ctx context.Context,
	topic string,
	group string,
) (queue.Message, bool, error) {
	endpoint := *c.baseURL
	endpoint.Path = path.Join(endpoint.Path, "/consume")

	query := endpoint.Query()
	query.Set("topic", topic)
	query.Set("group", group)
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return queue.Message{}, false, fmt.Errorf("build consume request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return queue.Message{}, false, fmt.Errorf("consume request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return queue.Message{}, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return queue.Message{}, false, unexpectedStatusError("consume request", resp)
	}

	var payload struct {
		Topic      string          `json:"topic"`
		ID         string          `json:"id"`
		RoutingKey string          `json:"routing_key"`
		Payload    json.RawMessage `json:"payload"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return queue.Message{}, false, fmt.Errorf("decode consume response: %w", err)
	}

	return queue.Message{
		Topic:      payload.Topic,
		ID:         payload.ID,
		RoutingKey: payload.RoutingKey,
		Payload:    []byte(payload.Payload),
	}, true, nil
}

func (c *HTTPClient) Ack(ctx context.Context, messageID string) error {
	endpoint := *c.baseURL
	endpoint.Path = path.Join(endpoint.Path, "/ack")

	body, err := json.Marshal(struct {
		MessageID string `json:"message_id"`
	}{MessageID: messageID})
	if err != nil {
		return fmt.Errorf("marshal ack request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint.String(),
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("build ack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("ack request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return unexpectedStatusError("ack request", resp)
	}

	return nil
}

func unexpectedStatusError(operation string, resp *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return fmt.Errorf("%s failed with status %d (read body: %w)", operation, resp.StatusCode, err)
	}

	if len(body) == 0 {
		return fmt.Errorf("%s failed with status %d", operation, resp.StatusCode)
	}

	return fmt.Errorf("%s failed with status %d: %s", operation, resp.StatusCode, bytes.TrimSpace(body))
}
