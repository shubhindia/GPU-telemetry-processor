package queue

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/shubhindia/gpu-telemetry/internal/logging"
)

type Runtime struct {
	cluster     Cluster
	partition   PartitionManager
	storage     Storage
	router      PartitionRouter
	replication map[int]*ReplicationCoordinator
	consumers   *consumerState
	metrics     *QueueMetrics
	logger      *slog.Logger
}

type consumerState struct {
	mu            sync.Mutex
	subscriptions map[string]*subscriptionState
	inflight      map[string]deliveryState
}

type subscriptionState struct {
	partitions map[int]*partitionConsumerState
}

type partitionConsumerState struct {
	nextOffset Offset
	inflight   *deliveryState
}

type deliveryState struct {
	messageID   string
	groupKey    string
	partitionID int
	offset      Offset
}

func NewRuntime(
	cluster Cluster,
	partition PartitionManager,
	storage Storage,
	router PartitionRouter,
	replication map[int]*ReplicationCoordinator,
) *Runtime {
	return &Runtime{
		cluster:     cluster,
		partition:   partition,
		storage:     storage,
		router:      router,
		replication: replication,
		metrics:     &QueueMetrics{},
		logger:      logging.Component("queue.runtime"),
		consumers: &consumerState{
			subscriptions: make(map[string]*subscriptionState),
			inflight:      make(map[string]deliveryState),
		},
	}
}

func (r *Runtime) Publish(
	ctx context.Context,
	topic string,
	message Message,
) error {
	message.Topic = topic

	partitionID, err := r.routePartition(topic, message)
	if err != nil {
		r.logger.Warn("route message", "topic", topic, "message_id", message.ID, "err", err)
		return err
	}

	if !r.partition.IsLeader(partitionID) {
		r.logger.Warn(
			"reject publish on follower",
			"topic", topic,
			"message_id", message.ID,
			"partition", partitionID,
		)
		return ErrNotPartitionLeader
	}

	if r.replication == nil {
		if _, err := r.storage.Append(
			ctx,
			partitionID,
			message,
		); err != nil {
			r.logger.Error("append message", "topic", topic, "message_id", message.ID, "partition", partitionID, "err", err)
			return err
		}
		r.metrics.Published.Add(1)
		r.logger.Debug("published message", "topic", topic, "message_id", message.ID, "partition", partitionID)
		return nil
	}

	coordinator := r.replication[partitionID]
	if coordinator == nil {
		if _, err := r.storage.Append(
			ctx,
			partitionID,
			message,
		); err != nil {
			r.logger.Error("append message", "topic", topic, "message_id", message.ID, "partition", partitionID, "err", err)
			return err
		}
		r.metrics.Published.Add(1)
		r.logger.Debug("published message", "topic", topic, "message_id", message.ID, "partition", partitionID)
		return nil
	}

	offset := r.partitionNextOffset(partitionID)

	if err := coordinator.Replicate(
		ctx,
		partitionID,
		Record{
			Offset:  offset,
			Message: message,
		},
	); err != nil {
		r.logger.Warn("replicate message", "topic", topic, "message_id", message.ID, "partition", partitionID, "err", err)
		return err
	}

	if _, err := r.storage.Append(
		ctx,
		partitionID,
		message,
	); err != nil {
		r.logger.Error("append message", "topic", topic, "message_id", message.ID, "partition", partitionID, "err", err)
		return err
	}

	r.metrics.Published.Add(1)
	r.logger.Debug("published message", "topic", topic, "message_id", message.ID, "partition", partitionID)
	return nil
}

func (r *Runtime) PartitionLeader(
	ctx context.Context,
	topic string,
	message Message,
) (Node, int, bool, error) {
	message.Topic = topic

	partitionID, err := r.routePartition(topic, message)
	if err != nil {
		return Node{}, 0, false, err
	}

	if r.partition.IsLeader(partitionID) {
		return r.partition.LocalNode(), partitionID, true, nil
	}

	leaderID, ok := r.partition.LeaderNodeID(partitionID)
	if !ok {
		return Node{}, partitionID, false, ErrLeaderNotFound
	}

	nodes, err := r.cluster.Nodes(ctx)
	if err != nil {
		return Node{}, partitionID, false, err
	}

	for _, node := range nodes {
		if node.ID == leaderID {
			return node, partitionID, false, nil
		}
	}

	return Node{}, partitionID, false, ErrLeaderNotFound
}

func (r *Runtime) routePartition(topic string, message Message) (int, error) {
	return r.router.Route(topic, message, r.partition.Partitions())
}

func (r *Runtime) Consume(
	ctx context.Context,
	topic string,
	group string,
) (<-chan Message, error) {
	if topic == "" {
		return nil, fmt.Errorf("topic is required")
	}
	if group == "" {
		return nil, fmt.Errorf("group is required")
	}

	out := make(chan Message)

	go func() {
		defer close(out)

		for {
			message, ok, err := r.Poll(ctx, topic, group)
			if err != nil {
				return
			}
			if !ok {
				if err := sleepContext(ctx, 100*time.Millisecond); err != nil {
					return
				}
				continue
			}

			select {
			case <-ctx.Done():
				return
			case out <- message:
			}
		}
	}()

	return out, nil
}

func (r *Runtime) Poll(
	ctx context.Context,
	topic string,
	group string,
) (Message, bool, error) {
	if topic == "" {
		return Message{}, false, fmt.Errorf("topic is required")
	}
	if group == "" {
		return Message{}, false, fmt.Errorf("group is required")
	}

	for _, partitionID := range r.leaderPartitionIDs() {
		message, ok, err := r.pollPartition(
			ctx,
			r.consumerKey(topic, group),
			topic,
			partitionID,
		)
		if err != nil {
			return Message{}, false, err
		}
		if ok {
			return message, true, nil
		}
	}

	return Message{}, false, nil
}

func (r *Runtime) Ack(ctx context.Context, messageID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := r.consumers.ack(messageID); err != nil {
		r.logger.Warn("ack message", "message_id", messageID, "err", err)
		return err
	}

	r.metrics.Acked.Add(1)
	r.logger.Debug("acked message", "message_id", messageID)
	return nil
}

func (r *Runtime) MetricsSnapshot() QueueMetricsSnapshot {
	metrics := r.metrics.Snapshot()
	metrics.Inflight = uint64(r.consumers.inflightCount())
	return metrics
}

func (r *Runtime) Stats() RuntimeStats {
	storage := make(map[int]partitionStoreStateSnapshot)
	if snapshotter, ok := r.storage.(interface{ Snapshot() partitionStoreSnapshot }); ok {
		for _, partition := range snapshotter.Snapshot().Partitions {
			storage[partition.ID] = partition
		}
	}

	partitions := r.partition.Partitions()
	sort.Slice(partitions, func(i int, j int) bool {
		return partitions[i].ID < partitions[j].ID
	})

	partitionStats := make([]RuntimePartitionStats, 0, len(partitions))
	for _, partition := range partitions {
		state := storage[partition.ID]
		partitionStats = append(partitionStats, RuntimePartitionStats{
			ID:             partition.ID,
			Role:           string(r.partitionRole(partition.ID)),
			NextOffset:     state.NextOffset,
			StoredMessages: uint64(state.NextOffset),
		})
	}

	return RuntimeStats{
		Counters:   r.MetricsSnapshot(),
		Partitions: partitionStats,
		Consumers:  r.consumers.snapshot(),
	}
}

func (r *Runtime) pollPartition(
	ctx context.Context,
	consumerKey string,
	topic string,
	partitionID int,
) (Message, bool, error) {
	for {
		offset, inFlight := r.consumers.nextOffset(consumerKey, partitionID)
		if inFlight {
			return Message{}, false, nil
		}

		message, err := r.storage.Read(ctx, partitionID, offset)
		if err != nil {
			if errors.Is(err, ErrOffsetNotFound) {
				return Message{}, false, nil
			}

			return Message{}, false, err
		}

		if message.Topic != topic {
			r.consumers.skip(consumerKey, partitionID, offset+1)
			continue
		}

		if !r.consumers.markInflight(
			consumerKey,
			partitionID,
			offset,
			message.ID,
		) {
			return Message{}, false, nil
		}

		r.metrics.Delivered.Add(1)
		r.logger.Debug(
			"delivered message",
			"topic", topic,
			"partition", partitionID,
			"offset", offset,
			"message_id", message.ID,
			"consumer", consumerKey,
		)

		return message, true, nil
	}
}

func (r *Runtime) leaderPartitionIDs() []int {
	partitions := r.partition.Partitions()
	ids := make([]int, 0, len(partitions))

	for _, partition := range partitions {
		if !r.partition.IsLeader(partition.ID) {
			continue
		}

		ids = append(ids, partition.ID)
	}

	sort.Ints(ids)
	return ids
}

func (r *Runtime) consumerKey(topic string, group string) string {
	return topic + "\x00" + group
}

func (r *Runtime) partitionRole(partitionID int) ReplicaRole {
	for _, partition := range r.partition.Partitions() {
		if partition.ID != partitionID {
			continue
		}

		for _, replica := range partition.Replicas {
			if replica.NodeID == r.partition.node.ID {
				return replica.Role
			}
		}
	}

	return ReplicaFollower
}

func (r *Runtime) partitionNextOffset(partitionID int) Offset {
	if snapshotter, ok := r.storage.(interface{ Snapshot() partitionStoreSnapshot }); ok {
		for _, partition := range snapshotter.Snapshot().Partitions {
			if partition.ID == partitionID {
				return partition.NextOffset
			}
		}
	}

	return 0
}

func (s *consumerState) nextOffset(groupKey string, partitionID int) (Offset, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	partition := s.ensurePartition(groupKey, partitionID)
	if partition.inflight != nil {
		return 0, true
	}

	return partition.nextOffset, false
}

func (s *consumerState) skip(groupKey string, partitionID int, next Offset) {
	s.mu.Lock()
	defer s.mu.Unlock()

	partition := s.ensurePartition(groupKey, partitionID)
	if partition.inflight != nil {
		return
	}

	partition.nextOffset = next
}

func (s *consumerState) markInflight(
	groupKey string,
	partitionID int,
	offset Offset,
	messageID string,
) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	partition := s.ensurePartition(groupKey, partitionID)
	if partition.inflight != nil {
		return false
	}

	delivery := deliveryState{
		messageID:   messageID,
		groupKey:    groupKey,
		partitionID: partitionID,
		offset:      offset,
	}

	partition.inflight = &delivery
	s.inflight[messageID] = delivery

	return true
}

func (s *consumerState) ack(messageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delivery, ok := s.inflight[messageID]
	if !ok {
		return ErrMessageNotInflight
	}

	partition := s.ensurePartition(delivery.groupKey, delivery.partitionID)
	if partition.inflight == nil || partition.inflight.messageID != messageID {
		delete(s.inflight, messageID)
		return ErrMessageNotInflight
	}

	partition.nextOffset = delivery.offset + 1
	partition.inflight = nil
	delete(s.inflight, messageID)

	return nil
}

func (s *consumerState) inflightCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.inflight)
}

func (s *consumerState) snapshot() []RuntimeConsumerStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	keys := make([]string, 0, len(s.subscriptions))
	for key := range s.subscriptions {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	stats := make([]RuntimeConsumerStats, 0, len(keys))
	for _, key := range keys {
		topic, group := splitConsumerKey(key)
		subscription := s.subscriptions[key]

		partitionIDs := make([]int, 0, len(subscription.partitions))
		for partitionID := range subscription.partitions {
			partitionIDs = append(partitionIDs, partitionID)
		}
		sort.Ints(partitionIDs)

		partitions := make([]RuntimeConsumerPartitionStats, 0, len(partitionIDs))
		for _, partitionID := range partitionIDs {
			partition := subscription.partitions[partitionID]
			partitions = append(partitions, RuntimeConsumerPartitionStats{
				PartitionID: partitionID,
				NextOffset:  partition.nextOffset,
				Inflight:    partition.inflight != nil,
			})
		}

		stats = append(stats, RuntimeConsumerStats{
			Topic:      topic,
			Group:      group,
			Partitions: partitions,
		})
	}

	return stats
}

func (s *consumerState) ensurePartition(
	groupKey string,
	partitionID int,
) *partitionConsumerState {
	subscription, ok := s.subscriptions[groupKey]
	if !ok {
		subscription = &subscriptionState{
			partitions: make(map[int]*partitionConsumerState),
		}
		s.subscriptions[groupKey] = subscription
	}

	partition, ok := subscription.partitions[partitionID]
	if !ok {
		partition = &partitionConsumerState{}
		subscription.partitions[partitionID] = partition
	}

	return partition
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func splitConsumerKey(key string) (string, string) {
	for i := 0; i < len(key); i++ {
		if key[i] == '\x00' {
			return key[:i], key[i+1:]
		}
	}

	return key, ""
}
