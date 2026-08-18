package telemetry

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/shubhindia/gpu-telemetry/internal/logging"
)

type Store struct {
	db     *sql.DB
	logger *slog.Logger
}

type Query struct {
	Start      time.Time
	End        time.Time
	MetricName string
	UUID       string
	Hostname   string
	GPUID      string
	Device     string
	Limit      int
}

type GPU struct {
	ID        string `json:"id"`
	GPUID     string `json:"gpu_id"`
	Device    string `json:"device"`
	UUID      string `json:"uuid"`
	ModelName string `json:"modelName"`
	Hostname  string `json:"Hostname"`
}

type SampleRecord struct {
	Timestamp  time.Time `json:"timestamp"`
	MetricName string    `json:"metric_name"`
	GPUID      string    `json:"gpu_id"`
	Device     string    `json:"device"`
	UUID       string    `json:"uuid"`
	ModelName  string    `json:"modelName"`
	Hostname   string    `json:"Hostname"`
	Container  string    `json:"container"`
	Pod        string    `json:"pod"`
	Namespace  string    `json:"namespace"`
	Value      float64   `json:"value"`
	LabelsRaw  string    `json:"labels_raw"`
}

func OpenStore(dataSourceName string) (*Store, error) {
	db, err := sql.Open("pgx", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("open postgres connection: %w", err)
	}

	store := &Store{
		db:     db,
		logger: logging.Component("telemetry.store"),
	}
	if err := store.init(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}

	store.logger.Info("telemetry store ready")

	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Insert(ctx context.Context, record Record) error {
	timestamp, err := time.Parse(time.RFC3339, record.Timestamp)
	if err != nil {
		return fmt.Errorf("parse telemetry timestamp: %w", err)
	}

	value, err := strconv.ParseFloat(record.Value, 64)
	if err != nil {
		return fmt.Errorf("parse telemetry value: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var seriesID int64
	err = tx.QueryRowContext(
		ctx,
		`INSERT INTO telemetry_series (
			metric_name, gpu_id, device, uuid, model_name, hostname,
			container_name, pod_name, namespace, labels_raw
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (
			metric_name, gpu_id, device, uuid, model_name, hostname,
			container_name, pod_name, namespace, labels_raw
		) DO UPDATE SET metric_name = EXCLUDED.metric_name
		RETURNING id`,
		record.MetricName,
		record.GPUID,
		record.Device,
		record.UUID,
		record.ModelName,
		record.Hostname,
		record.Container,
		record.Pod,
		record.Namespace,
		record.LabelsRaw,
	).Scan(&seriesID)
	if err != nil {
		return fmt.Errorf("upsert series: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO telemetry_samples (series_id, recorded_at, value) VALUES ($1, $2, $3)`,
		seriesID,
		timestamp.UTC(),
		value,
	); err != nil {
		return fmt.Errorf("insert sample: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	s.logger.Debug(
		"inserted telemetry sample",
		"metric_name", record.MetricName,
		"uuid", record.UUID,
		"gpu_id", record.GPUID,
		"timestamp", timestamp.UTC(),
	)

	return nil
}

func (s *Store) Query(ctx context.Context, query Query) ([]SampleRecord, error) {
	if !query.End.IsZero() && !query.Start.IsZero() && query.End.Before(query.Start) {
		return nil, fmt.Errorf("end time must not be before start time")
	}

	sqlQuery, args := buildQuery(query)
	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query telemetry: %w", err)
	}
	defer rows.Close()

	results := make([]SampleRecord, 0)
	for rows.Next() {
		var record SampleRecord
		if err := rows.Scan(
			&record.Timestamp,
			&record.MetricName,
			&record.GPUID,
			&record.Device,
			&record.UUID,
			&record.ModelName,
			&record.Hostname,
			&record.Container,
			&record.Pod,
			&record.Namespace,
			&record.Value,
			&record.LabelsRaw,
		); err != nil {
			return nil, fmt.Errorf("scan telemetry row: %w", err)
		}

		results = append(results, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate telemetry rows: %w", err)
	}

	s.logger.Debug(
		"queried telemetry samples",
		"start", query.Start.UTC(),
		"end", query.End.UTC(),
		"metric_name", query.MetricName,
		"uuid", query.UUID,
		"hostname", query.Hostname,
		"gpu_id", query.GPUID,
		"device", query.Device,
		"limit", query.Limit,
		"results", len(results),
	)

	return results, nil
}

func (s *Store) ListGPUs(ctx context.Context) ([]GPU, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT DISTINCT uuid, gpu_id, device, model_name, hostname
		 FROM telemetry_series
		 ORDER BY hostname ASC, gpu_id ASC, uuid ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list gpus: %w", err)
	}
	defer rows.Close()

	results := make([]GPU, 0)
	for rows.Next() {
		var gpu GPU
		if err := rows.Scan(
			&gpu.UUID,
			&gpu.GPUID,
			&gpu.Device,
			&gpu.ModelName,
			&gpu.Hostname,
		); err != nil {
			return nil, fmt.Errorf("scan gpu row: %w", err)
		}

		gpu.ID = gpu.UUID
		results = append(results, gpu)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate gpu rows: %w", err)
	}

	s.logger.Debug("listed gpus", "results", len(results))

	return results, nil
}

func (s *Store) init(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS telemetry_series (
			id BIGSERIAL PRIMARY KEY,
			metric_name TEXT NOT NULL,
			gpu_id TEXT NOT NULL,
			device TEXT NOT NULL,
			uuid TEXT NOT NULL,
			model_name TEXT NOT NULL,
			hostname TEXT NOT NULL,
			container_name TEXT NOT NULL,
			pod_name TEXT NOT NULL,
			namespace TEXT NOT NULL,
			labels_raw TEXT NOT NULL,
			UNIQUE (
				metric_name, gpu_id, device, uuid, model_name, hostname,
				container_name, pod_name, namespace, labels_raw
			)
		)`,
		`CREATE TABLE IF NOT EXISTS telemetry_samples (
			id BIGSERIAL PRIMARY KEY,
			series_id BIGINT NOT NULL REFERENCES telemetry_series(id) ON DELETE CASCADE,
			recorded_at TIMESTAMPTZ NOT NULL,
			value DOUBLE PRECISION NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS telemetry_samples_recorded_at_idx ON telemetry_samples (recorded_at)`,
		`CREATE INDEX IF NOT EXISTS telemetry_samples_series_recorded_at_idx ON telemetry_samples (series_id, recorded_at)`,
		`CREATE INDEX IF NOT EXISTS telemetry_series_metric_name_idx ON telemetry_series (metric_name)`,
		`CREATE INDEX IF NOT EXISTS telemetry_series_uuid_idx ON telemetry_series (uuid)`,
		`CREATE INDEX IF NOT EXISTS telemetry_series_hostname_idx ON telemetry_series (hostname)`,
	} {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("initialize telemetry store: %w", err)
		}
	}

	return nil
}

func buildQuery(query Query) (string, []any) {
	clauses := []string{"ts.recorded_at >= $1", "ts.recorded_at <= $2"}
	args := []any{query.Start.UTC(), query.End.UTC()}

	addFilter := func(column string, value string) {
		if value == "" {
			return
		}

		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	addFilter("sr.metric_name", query.MetricName)
	addFilter("sr.uuid", query.UUID)
	addFilter("sr.hostname", query.Hostname)
	addFilter("sr.gpu_id", query.GPUID)
	addFilter("sr.device", query.Device)

	statement := strings.Builder{}
	statement.WriteString(`SELECT
		ts.recorded_at,
		 sr.metric_name,
		 sr.gpu_id,
		 sr.device,
		 sr.uuid,
		 sr.model_name,
		 sr.hostname,
		 sr.container_name,
		 sr.pod_name,
		 sr.namespace,
		 ts.value,
		 sr.labels_raw
	FROM telemetry_samples ts
	JOIN telemetry_series sr ON sr.id = ts.series_id
	WHERE `)
	statement.WriteString(strings.Join(clauses, " AND "))
	statement.WriteString(" ORDER BY ts.recorded_at ASC")

	if query.Limit > 0 {
		args = append(args, query.Limit)
		statement.WriteString(fmt.Sprintf(" LIMIT $%d", len(args)))
	}

	return statement.String(), args
}
