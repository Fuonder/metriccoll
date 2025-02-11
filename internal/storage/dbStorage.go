package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/logger"
	"github.com/Fuonder/metriccoll.git/internal/models"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
	"sync"
	"time"
)

//type DBFileStoreInfo struct {
//	Sync          bool
//	StoreInterval time.Duration
//	fStore        bool
//	fStoragePath  string
//}

// Database TODO: write tests
type Database struct {
	connection *sql.DB
	settings   string
	//fileStoreInfo DBFileStoreInfo // fileStoreInfo TODO: if it will be necessary to load/save files
	//fileMu        sync.RWMutex // fileMu TODO: consider necessity
	rwMutex sync.RWMutex
}

func NewDatabase(settings string) (*Database, error) {
	var err error
	logger.Log.Info("Connecting to database")
	db := Database{settings: settings}
	db.connection, err = sql.Open("pgx", db.settings)
	if err != nil {
		return &Database{}, fmt.Errorf("can not create new database: %v", err)
	}
	logger.Log.Info("Database initial connection successful")
	err = db.CheckConnection()
	if err != nil {
		return &Database{}, fmt.Errorf("access to database: %v", err)
	}
	return &db, nil
}

func (db *Database) CreateTables() error {
	logger.Log.Info("Creating tables in database")
	query := `
	CREATE TABLE IF NOT EXISTS gauge_metrics (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		value DOUBLE PRECISION
	);
	`
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.connection.ExecContext(ctx, query)
	if err != nil {
		logger.Log.Fatal("Failed to create gauge table", zap.Error(err))
		return err
	}

	query = `
	CREATE TABLE IF NOT EXISTS counter_metrics (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		delta integer
	);
	`
	_, err = db.connection.ExecContext(ctx, query)
	if err != nil {
		logger.Log.Fatal("Failed to create counter table", zap.Error(err))
		return err
	}
	logger.Log.Info("Tables created successfully")
	return nil
}

func (db *Database) CheckConnection() error {
	logger.Log.Info("Checking db connection")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if db.connection == nil {
		logger.Log.Warn("no active connection with db")
		return fmt.Errorf("no active connection with db")
	}
	if err := db.connection.PingContext(ctx); err != nil {
		return fmt.Errorf("can not ping database: %v", err)
	}
	logger.Log.Info("Connection - OK")
	return nil
}

func (db *Database) Close() error {
	logger.Log.Info("Closing database connection gracefully")
	if db.connection != nil {
		return db.connection.Close()
	}
	return nil
}

func (db *Database) GetMetricByName(name string, mType string) (models.Metrics, error) {
	var (
		query  string
		metric models.Metrics
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if db.connection == nil {
		return models.Metrics{}, fmt.Errorf("no active connection with db")
	}
	if mType == "gauge" {
		query = `SELECT id, type, value from gauge_metrics WHERE id = $1`
		err := db.connection.QueryRowContext(ctx, query, name).Scan(&metric.ID, &metric.MType, &metric.Value)
		if err != nil {
			if err == sql.ErrNoRows {
				return models.Metrics{}, fmt.Errorf("metric not found for id '%s' and type '%s'", name, mType)
			}
			return models.Metrics{}, fmt.Errorf("error searching for metric: %v", err)
		}
	} else if mType == "counter" {
		query = `SELECT id, type, delta from counter_metrics WHERE id = $1`
		err := db.connection.QueryRowContext(ctx, query, name).Scan(&metric.ID, &metric.MType, &metric.Delta)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return models.Metrics{}, fmt.Errorf("metric not found for id '%s' and type '%s'", name, mType)
			}
			return models.Metrics{}, fmt.Errorf("error searching for metric: %v", err)
		}
	} else {
		return models.Metrics{}, fmt.Errorf("metric type: %s is not supported", mType)
	}

	return metric, nil
}

func (db *Database) AppendMetric(metric models.Metrics) error {
	db.rwMutex.Lock()
	defer db.rwMutex.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if db.connection == nil {
		return fmt.Errorf("no active connection with db")
	}
	if metric.MType == "gauge" {
		if metric.Value == nil {
			return ErrInvalidMetricValue
		}
		query := `
			INSERT INTO gauge_metrics (id, type, value)
			VALUES ($1, 'gauge', $2)
			ON CONFLICT (id)
			DO UPDATE SET value = EXCLUDED.value;
		`
		_, err := db.connection.ExecContext(ctx, query, metric.ID, metric.Value)
		if err != nil {
			return fmt.Errorf("can not append gaguge metric: %v", err)
		}
		return nil
	} else if metric.MType == "counter" {
		if metric.Delta == nil {
			return ErrInvalidMetricValue
		}
		query := `
			INSERT INTO counter_metrics (id, type, delta)
			VALUES ($1, 'counter', $2)
			ON CONFLICT (id)
			DO UPDATE SET delta = counter_metrics.delta + EXCLUDED.delta;
		`
		_, err := db.connection.ExecContext(ctx, query, metric.ID, metric.Delta)
		if err != nil {
			return fmt.Errorf("can not append counter metric: %v", err)
		}
		return nil
	} else {
		return fmt.Errorf("metric type: %s is not supported", metric.MType)
	}
}

// GetAllMetrics TODO: error handling
func (db *Database) GetAllMetrics() []models.Metrics {
	var metrics []models.Metrics
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if db.connection == nil {
		return nil
	}
	rowsCounter, err := db.connection.QueryContext(ctx, `SELECT id, type, delta FROM counter_metrics`)
	if err != nil {
		return nil
	}
	defer rowsCounter.Close()

	for rowsCounter.Next() {
		var m models.Metrics
		if err := rowsCounter.Scan(&m.ID, &m.MType, &m.Delta); err != nil {
			return nil
		}
		metrics = append(metrics, m)
	}
	if err := rowsCounter.Err(); err != nil {
		return nil
	}

	rowsGauge, err := db.connection.QueryContext(ctx, `SELECT id, type, value FROM gauge_metrics`)
	if err != nil {
		return nil
	}
	defer rowsGauge.Close()
	for rowsGauge.Next() {
		var m models.Metrics
		if err := rowsGauge.Scan(&m.ID, &m.MType, &m.Value); err != nil {
			return nil
		}
		metrics = append(metrics, m)
	}
	if err := rowsGauge.Err(); err != nil {
		return nil
	}
	return metrics
}

func (db *Database) DumpMetrics() error {
	return fmt.Errorf("dump to file is not yet implemented, " +
		"consider using \"jsonStorage\"")
}

func (db *Database) loadMetricsFromFile() error {
	return fmt.Errorf("loading from file is not yet implemented" +
		"consider using \"jsonStorage\"")
}
