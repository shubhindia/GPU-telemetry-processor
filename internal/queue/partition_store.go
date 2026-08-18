package queue

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type PartitionStore struct {
	mu sync.RWMutex

	dataDir string
	config  QueueConfig

	segments map[int][]*SegmentStore
	states   map[int]*partitionState
}

type partitionState struct {
	currentSegmentID uint64
	nextOffset       Offset
}

type partitionStoreSnapshot struct {
	Partitions []partitionStoreStateSnapshot
}

type partitionStoreStateSnapshot struct {
	ID         int
	NextOffset Offset
}

type QueueConfig struct {
	SegmentSizeBytes     int64
	ReplicationFactor    int
	RequiredFollowerAcks int
}

func NewPartitionStore(dataDir string, config QueueConfig) *PartitionStore {
	return &PartitionStore{
		dataDir:  dataDir,
		config:   config,
		segments: make(map[int][]*SegmentStore),
		states:   make(map[int]*partitionState),
	}
}

func (s *PartitionStore) OpenPartition(partitionID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.states[partitionID]; exists {
		return nil
	}

	if err := os.MkdirAll(s.dataDir, 0750); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	segmentPath := s.segmentPath(partitionID, 0)

	segment, err := OpenSegmentStore(segmentPath)
	if err != nil {
		return fmt.Errorf("open partition segment: %w", err)
	}

	nextOffset, err := segment.Recover()
	if err != nil {
		_ = segment.Close()
		return fmt.Errorf("recover partition segment: %w", err)
	}

	s.segments[partitionID] = []*SegmentStore{segment}
	s.states[partitionID] = &partitionState{
		currentSegmentID: 0,
		nextOffset:       nextOffset,
	}

	return nil
}

func (s *PartitionStore) Append(
	ctx context.Context,
	partitionID int,
	message Message,
) (Offset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return 0, err
	}

	state, exists := s.states[partitionID]
	if !exists {
		return 0, fmt.Errorf("partition %d is not open", partitionID)
	}

	segment := s.segments[partitionID][state.currentSegmentID]

	offset := state.nextOffset

	if err := segment.Append(Record{
		Offset:  offset,
		Message: message,
	}); err != nil {
		return 0, fmt.Errorf("append partition %d: %w", partitionID, err)
	}

	if err := segment.Flush(); err != nil {
		return 0, fmt.Errorf("flush partition %d: %w", partitionID, err)
	}

	state.nextOffset++

	return offset, nil
}

func (s *PartitionStore) Read(
	ctx context.Context,
	partitionID int,
	offset Offset,
) (Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := ctx.Err(); err != nil {
		return Message{}, err
	}

	segments, exists := s.segments[partitionID]
	if !exists {
		return Message{}, fmt.Errorf("partition %d is not open", partitionID)
	}

	for _, segment := range segments {
		record, err := segment.Read(offset)
		if err == nil {
			return record.Message, nil
		}

		if err == ErrOffsetNotFound {
			continue
		}

		return Message{}, fmt.Errorf(
			"read partition %d offset %d: %w",
			partitionID,
			offset,
			err,
		)
	}

	return Message{}, fmt.Errorf(
		"partition %d offset %d: %w",
		partitionID,
		offset,
		ErrOffsetNotFound,
	)
}

func (s *PartitionStore) Flush(
	ctx context.Context,
	partitionID int,
) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := ctx.Err(); err != nil {
		return err
	}

	segments, exists := s.segments[partitionID]
	if !exists {
		return fmt.Errorf("partition %d is not open", partitionID)
	}

	for _, segment := range segments {
		if err := segment.Flush(); err != nil {
			return fmt.Errorf(
				"flush partition %d: %w",
				partitionID,
				err,
			)
		}
	}

	return nil
}

func (s *PartitionStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var firstErr error

	for partitionID, segments := range s.segments {
		for _, segment := range segments {
			if err := segment.Close(); err != nil && firstErr == nil {
				firstErr = fmt.Errorf(
					"close partition %d: %w",
					partitionID,
					err,
				)
			}
		}
	}

	s.segments = make(map[int][]*SegmentStore)
	s.states = make(map[int]*partitionState)

	return firstErr
}

func (s *PartitionStore) Snapshot() partitionStoreSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	partitions := make([]partitionStoreStateSnapshot, 0, len(s.states))
	for partitionID, state := range s.states {
		partitions = append(partitions, partitionStoreStateSnapshot{
			ID:         partitionID,
			NextOffset: state.nextOffset,
		})
	}

	return partitionStoreSnapshot{Partitions: partitions}
}

func (s *PartitionStore) AppendRecord(
	ctx context.Context,
	partitionID int,
	record Record,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}

	state, exists := s.states[partitionID]
	if !exists {
		return fmt.Errorf("partition %d is not open", partitionID)
	}

	if record.Offset != state.nextOffset {
		return fmt.Errorf(
			"%w for partition %d: expected %d, got %d",
			ErrUnexpectedOffset,
			partitionID,
			state.nextOffset,
			record.Offset,
		)
	}

	segment := s.segments[partitionID][state.currentSegmentID]

	if err := segment.Append(record); err != nil {
		return fmt.Errorf(
			"append record to partition %d: %w",
			partitionID,
			err,
		)
	}

	if err := segment.Flush(); err != nil {
		return fmt.Errorf(
			"flush partition %d: %w",
			partitionID,
			err,
		)
	}

	state.nextOffset++

	return nil
}

func (s *PartitionStore) segmentPath(
	partitionID int,
	segmentID uint64,
) string {
	return filepath.Join(
		s.dataDir,
		fmt.Sprintf(
			"partition-%06d-%06d.log",
			partitionID,
			segmentID,
		),
	)
}
