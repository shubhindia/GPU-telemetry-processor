package queue

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestSegmentStorePersistsMessageAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "segment.log")

	original := Message{
		ID:         "message-1",
		RoutingKey: "GPU-123",
		Payload:    []byte(`{"metric":"gpu_util","value":95}`),
	}

	// First process: append and flush.
	store, err := OpenSegmentStore(path)
	if err != nil {
		t.Fatalf("open segment: %v", err)
	}

	if err := store.Append(Record{
		Offset:  42,
		Message: original,
	}); err != nil {
		t.Fatalf("append record: %v", err)
	}

	if err := store.Flush(); err != nil {
		t.Fatalf("flush segment: %v", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close segment: %v", err)
	}

	// Simulate process restart: reopen the same segment.
	store, err = OpenSegmentStore(path)
	if err != nil {
		t.Fatalf("reopen segment: %v", err)
	}
	defer store.Close()

	record, err := store.Read(42)
	if err != nil {
		t.Fatalf("read record after restart: %v", err)
	}

	if record.Offset != 42 {
		t.Fatalf("expected offset 42, got %d", record.Offset)
	}

	if record.Message.ID != original.ID {
		t.Fatalf("expected message ID %q, got %q", original.ID, record.Message.ID)
	}

	if record.Message.RoutingKey != original.RoutingKey {
		t.Fatalf(
			"expected routing key %q, got %q",
			original.RoutingKey,
			record.Message.RoutingKey,
		)
	}

	if string(record.Message.Payload) != string(original.Payload) {
		t.Fatalf(
			"expected payload %q, got %q",
			original.Payload,
			record.Message.Payload,
		)
	}
}

func TestSegmentStoreRecoversFromPartialRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "segment.log")

	store, err := OpenSegmentStore(path)
	if err != nil {
		t.Fatalf("open segment: %v", err)
	}

	message := Message{
		ID:         "message-1",
		RoutingKey: "GPU-123",
		Payload:    []byte("payload"),
	}

	if err := store.Append(Record{
		Offset:  10,
		Message: message,
	}); err != nil {
		t.Fatalf("append record: %v", err)
	}

	if err := store.Flush(); err != nil {
		t.Fatalf("flush segment: %v", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close segment: %v", err)
	}

	// Simulate a crash during the next write by appending
	// an incomplete record directly to the file.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		t.Fatalf("open segment for corruption: %v", err)
	}

	header := make([]byte, recordHeaderSize)
	binary.BigEndian.PutUint32(header[0:4], 100)
	binary.BigEndian.PutUint64(header[4:12], 11)

	if _, err := file.Write(header); err != nil {
		t.Fatalf("write partial header: %v", err)
	}

	if _, err := file.Write([]byte("partial")); err != nil {
		t.Fatalf("write partial payload: %v", err)
	}

	if err := file.Close(); err != nil {
		t.Fatalf("close corrupted segment: %v", err)
	}

	store, err = OpenSegmentStore(path)
	if err != nil {
		t.Fatalf("reopen segment: %v", err)
	}
	defer store.Close()

	nextOffset, err := store.Recover()
	if err != nil {
		t.Fatalf("recover segment: %v", err)
	}

	if nextOffset != 11 {
		t.Fatalf("expected next offset 11, got %d", nextOffset)
	}

	record, err := store.Read(10)
	if err != nil {
		t.Fatalf("read recovered record: %v", err)
	}

	if record.Message.ID != message.ID {
		t.Fatalf("expected message ID %q, got %q", message.ID, record.Message.ID)
	}

	if _, err := store.Read(11); err != ErrOffsetNotFound {
		t.Fatalf("expected offset 11 to be unavailable, got %v", err)
	}
}
