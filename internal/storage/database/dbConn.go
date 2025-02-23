package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/logger"
	"github.com/Fuonder/metriccoll.git/internal/models"
	"github.com/Fuonder/metriccoll.git/internal/storage"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
	"strings"
	"time"
)

func isConnectionError(err error) bool {
	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		state := pgErr.SQLState()
		if strings.HasPrefix(state, "08") {
			return true
		}
	}
	return false
}

type PSQLConnection struct {
	db *sql.DB
}

func NewPSQLConnection(ctx context.Context, settings string) (*PSQLConnection, error) {
	var err error
	c := &PSQLConnection{}

	logger.Log.Info("Connecting to database")
	c.db, err = sql.Open("pgx", settings)
	if err != nil {
		return &PSQLConnection{}, fmt.Errorf("can not connect with database: %v", err)
	}
	logger.Log.Info("Database initial connection successful")
	err = c.TryConnectContext(ctx)
	if err != nil {
		return &PSQLConnection{}, fmt.Errorf("access to database: %v", err)
	}
	return c, nil
}

func (c *PSQLConnection) TryConnectContext(ctx context.Context) error {
	var err error
	logger.Log.Info("Checking db accessibility")
	if c.db == nil {
		logger.Log.Warn("no active connection with db")
		return fmt.Errorf("no active connection with db")
	}
	for i := 0; i < maxRetries; i++ {
		err = c.db.PingContext(ctx)
		if err == nil {
			logger.Log.Info("Access - OK")
			return nil
		} else if isConnectionError(err) {
			if i < len(timeouts) {
				logger.Log.Info("can not access database", zap.Error(err))
				logger.Log.Info("retrying after timeout",
					zap.Duration("timeout", timeouts[i]),
					zap.Int("retry-count", i+1))
				time.Sleep(timeouts[i])
			}
		} else {
			return fmt.Errorf("can not access database: %v", err)
		}
	}
	return fmt.Errorf("can not access database: %v", err)
}

func (c *PSQLConnection) CreateTablesContext(ctx context.Context) error {
	logger.Log.Info("Creating tables in database")
	query := `
			CREATE TABLE IF NOT EXISTS gauge_metrics (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			value DOUBLE PRECISION);
		`

	_, err := c.db.ExecContext(ctx, query)
	if err != nil {
		logger.Log.Fatal("Failed to create gauge table", zap.Error(err))
		return err
	}

	query = `
			CREATE TABLE IF NOT EXISTS counter_metrics (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			delta BIGINT);
			`
	_, err = c.db.ExecContext(ctx, query)
	if err != nil {
		logger.Log.Fatal("Failed to create counter table", zap.Error(err))
		return err
	}
	logger.Log.Info("Tables created successfully")
	return nil

}

func (c *PSQLConnection) GetGaugeMetric(ctx context.Context, name string) (models.Metrics, error) {
	var (
		query  string
		metric models.Metrics
	)

	query = `SELECT id, type, value from gauge_metrics WHERE id = $1`
	err := c.db.QueryRowContext(ctx, query, name).Scan(&metric.ID, &metric.MType, &metric.Value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Metrics{}, fmt.Errorf("metric not found for id '%s' and type 'Gauge'", name)
		}
		return models.Metrics{}, fmt.Errorf("error searching for metric: %v", err)
	}
	return metric, nil

}

func (c *PSQLConnection) GetCounterMetric(ctx context.Context, name string) (models.Metrics, error) {
	var (
		query  string
		metric models.Metrics
	)

	query = `SELECT id, type, delta from counter_metrics WHERE id = $1`
	err := c.db.QueryRowContext(ctx, query, name).Scan(&metric.ID, &metric.MType, &metric.Delta)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Metrics{}, fmt.Errorf("metric not found for id '%s' and type 'Counter'", name)
		}
		return models.Metrics{}, fmt.Errorf("error searching for metric: %v", err)
	}
	return metric, nil
}

func (c *PSQLConnection) AppendGaugeMetric(ctx context.Context, metric models.Metrics) error {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
			INSERT INTO gauge_metrics (id, type, value)
			VALUES ($1, 'gauge', $2)
			ON CONFLICT (id)
			DO UPDATE SET value = EXCLUDED.value;
		`
	_, err = tx.ExecContext(ctx, query, metric.ID, metric.Value)
	if err != nil {
		return fmt.Errorf("can not append gaguge metric: %v", err)
	}
	return tx.Commit()

}
func (c *PSQLConnection) AppendCounterMetric(ctx context.Context, metric models.Metrics) error {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
			INSERT INTO counter_metrics (id, type, delta)
			VALUES ($1, 'counter', $2)
			ON CONFLICT (id)
			DO UPDATE SET delta = counter_metrics.delta + EXCLUDED.delta;
		`
	_, err = tx.ExecContext(ctx, query, metric.ID, metric.Delta)
	if err != nil {
		return fmt.Errorf("can not append counter metric: %v", err)
	}
	return tx.Commit()
}

func (c *PSQLConnection) GetAllMetrics(ctx context.Context) ([]models.Metrics, error) {
	var metrics []models.Metrics
	queryGauge := `SELECT id, type, delta FROM counter_metrics`
	queryCounter := `SELECT id, type, value FROM gauge_metrics`

	rowsCounter, err := c.db.QueryContext(ctx, queryCounter)
	if err != nil {
		return []models.Metrics{}, err
	}
	defer rowsCounter.Close()
	for rowsCounter.Next() {
		var m models.Metrics
		if err := rowsCounter.Scan(&m.ID, &m.MType, &m.Delta); err != nil {
			return []models.Metrics{}, err
		}
		metrics = append(metrics, m)
	}
	if err := rowsCounter.Err(); err != nil {
		return []models.Metrics{}, err
	}

	rowsGauge, err := c.db.QueryContext(ctx, queryGauge)
	if err != nil {
		return []models.Metrics{}, err
	}
	defer rowsGauge.Close()
	for rowsGauge.Next() {
		var m models.Metrics
		if err := rowsGauge.Scan(&m.ID, &m.MType, &m.Value); err != nil {
			return []models.Metrics{}, err
		}
		metrics = append(metrics, m)
	}
	if err := rowsGauge.Err(); err != nil {
		return []models.Metrics{}, err
	}
	return metrics, nil
}

func (c *PSQLConnection) AppendBatch(ctx context.Context, metrics []models.Metrics) error {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmtGauge, err := tx.PrepareContext(ctx,
		`
			INSERT INTO gauge_metrics (id, type, value)
			VALUES ($1, 'gauge', $2)
			ON CONFLICT (id)
			DO UPDATE SET value = EXCLUDED.value;
		`)
	if err != nil {
		return err
	}
	defer stmtGauge.Close()
	stmtCounter, err := tx.PrepareContext(ctx,
		`
			INSERT INTO counter_metrics (id, type, delta)
			VALUES ($1, 'counter', $2)
			ON CONFLICT (id)
			DO UPDATE SET delta = counter_metrics.delta + EXCLUDED.delta;
		`)
	if err != nil {
		return err
	}
	defer stmtCounter.Close()

	for _, m := range metrics {
		if m.MType == "gauge" {
			if m.Value == nil {
				return storage.ErrInvalidMetricValue
			}
			_, err = stmtGauge.ExecContext(ctx,
				m.ID,
				m.Value)
			if err != nil {
				return err
			}
		} else if m.MType == "counter" {
			if m.Delta == nil {
				return storage.ErrInvalidMetricValue
			}
			_, err = stmtCounter.ExecContext(ctx,
				m.ID,
				m.Delta)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("metric type: %s is not supported", m.MType)
		}
	}
	return tx.Commit()
}

func (c *PSQLConnection) Close() error {
	return c.db.Close()
}
