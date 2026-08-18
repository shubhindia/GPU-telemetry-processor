package queue

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
)

const recordHeaderSize = 12

type SegmentStore struct {
	file *os.File
	mu   sync.RWMutex
}

func OpenSegmentStore(path string) (*SegmentStore, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("open segment: %w", err)
	}

	return &SegmentStore{
		file: file,
	}, nil
}

func (s *SegmentStore) Append(record Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	payload, err := encodeMessage(record.Message)
	if err != nil {
		return fmt.Errorf("encode record: %w", err)
	}

	header := make([]byte, recordHeaderSize)
	binary.BigEndian.PutUint32(header[0:4], uint32(len(payload)))
	binary.BigEndian.PutUint64(header[4:12], uint64(record.Offset))

	if _, err := s.file.Write(header); err != nil {
		return fmt.Errorf("write record header: %w", err)
	}

	if _, err := s.file.Write(payload); err != nil {
		return fmt.Errorf("write record payload: %w", err)
	}

	return nil
}

func (s *SegmentStore) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.file.Sync(); err != nil {
		return fmt.Errorf("sync segment: %w", err)
	}

	return nil
}

func (s *SegmentStore) Read(offset Offset) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.file.Seek(0, io.SeekStart); err != nil {
		return Record{}, fmt.Errorf("seek segment: %w", err)
	}

	header := make([]byte, recordHeaderSize)

	for {
		if _, err := io.ReadFull(s.file, header); err != nil {
			if err == io.EOF {
				return Record{}, ErrOffsetNotFound
			}

			return Record{}, fmt.Errorf("read record header: %w", err)

		}

		payloadSize := binary.BigEndian.Uint32(header[0:4])
		recordOffset := Offset(binary.BigEndian.Uint64(header[4:12]))

		payload := make([]byte, payloadSize)

		if _, err := io.ReadFull(s.file, payload); err != nil {
			return Record{}, fmt.Errorf("read record payload: %w", err)
		}

		if recordOffset != offset {
			continue
		}

		message, err := decodeMessage(payload)
		if err != nil {
			return Record{}, fmt.Errorf("decode record: %w", err)
		}

		return Record{
			Offset:  recordOffset,
			Message: message,
		}, nil
	}
}

func (s *SegmentStore) Recover() (Offset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.file.Seek(0, io.SeekStart); err != nil {
		return 0, fmt.Errorf("seek segment: %w", err)
	}

	header := make([]byte, recordHeaderSize)

	var (
		nextOffset Offset
		validBytes int64
	)

	for {
		_, err := io.ReadFull(s.file, header)
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		payloadSize := binary.BigEndian.Uint32(header[0:4])
		recordOffset := Offset(binary.BigEndian.Uint64(header[4:12]))

		payload := make([]byte, payloadSize)

		if _, err := io.ReadFull(s.file, payload); err != nil {
			break
		}

		if _, err := decodeMessage(payload); err != nil {
			break
		}

		validBytes += int64(recordHeaderSize) + int64(payloadSize)

		if recordOffset >= nextOffset {
			nextOffset = recordOffset + 1
		}
	}

	if err := s.file.Truncate(validBytes); err != nil {
		return 0, fmt.Errorf("truncate invalid segment tail: %w", err)
	}

	if _, err := s.file.Seek(0, io.SeekEnd); err != nil {
		return 0, fmt.Errorf("seek segment end: %w", err)
	}

	return nextOffset, nil
}

func (s *SegmentStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.file.Close(); err != nil {
		return fmt.Errorf("close segment: %w", err)
	}

	return nil
}
