package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type HTTPReplicationTransport struct {
	client *http.Client
	path   string
}

func NewHTTPReplicationTransport(
	client *http.Client,
	path string,
) *HTTPReplicationTransport {
	if client == nil {
		client = &http.Client{}
	}

	return &HTTPReplicationTransport{
		client: client,
		path:   path,
	}
}

type replicationRequest struct {
	PartitionID int    `json:"partition_id"`
	Record      Record `json:"record"`
}

func (t *HTTPReplicationTransport) Replicate(
	ctx context.Context,
	node Node,
	partitionID int,
	record Record,
) error {
	payload, err := json.Marshal(replicationRequest{
		PartitionID: partitionID,
		Record:      record,
	})
	if err != nil {
		return fmt.Errorf("encode replication request: %w", err)
	}

	url := strings.TrimRight(node.Address, "/") + "/" + strings.TrimLeft(t.path, "/")

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewReader(payload),
	)
	if err != nil {
		return fmt.Errorf("create replication request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("send replication request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = http.StatusText(resp.StatusCode)
		}

		return fmt.Errorf(
			"replication request to %s failed: %s: %s",
			url,
			resp.Status,
			message,
		)
	}

	return nil
}
