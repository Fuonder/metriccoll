package database

import (
	"context"
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/logger"
	"github.com/Fuonder/metriccoll.git/internal/models"
	"github.com/Fuonder/metriccoll.git/internal/storage"
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

var timeouts = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}
var maxRetries = 3

// Database TODO: write tests
type DBStorage struct {
	//connection *sql.DB
	connection storage.DBConnection
	//settings   string
	//fileStoreInfo DBFileStoreInfo // fileStoreInfo TODO: if it will be necessary to load/save files
	//fileMu        sync.RWMutex // fileMu TODO: consider necessity
	rwMutex sync.RWMutex
}

func NewDBStorage(ctx context.Context, conn storage.DBConnection) (*DBStorage, error) {
	var (
		err error
	)
	db := &DBStorage{connection: conn}
	err = db.connection.TryConnectContext(ctx)
	if err != nil {
		return &DBStorage{}, fmt.Errorf("NewDBStorage: %v", err)
	}
	return db, nil
}

func (db *DBStorage) CheckConnection() error {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = db.connection.TryConnectContext(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (db *DBStorage) Close() error {
	logger.Log.Info("Closing database connection gracefully")
	if db.connection != nil {
		return db.connection.Close()
	}
	return nil
}

func (db *DBStorage) GetMetricByName(name string, mType string) (models.Metrics, error) {
	var (
		metric models.Metrics
		err    error
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if db.connection == nil {
		return models.Metrics{}, fmt.Errorf("no active connection with db")
	}
	if mType == "gauge" {
		metric, err = db.connection.GetGaugeMetric(ctx, name)
		if err != nil {
			return models.Metrics{}, fmt.Errorf("GetMetricByName: %v", err)
		}
	} else if mType == "counter" {
		metric, err = db.connection.GetCounterMetric(ctx, name)
		if err != nil {
			return models.Metrics{}, fmt.Errorf("GetMetricByName: %v", err)
		}
	} else {
		return models.Metrics{}, fmt.Errorf("metric type: %s is not supported", mType)
	}
	return metric, nil
}

func (db *DBStorage) AppendMetric(metric models.Metrics) error {
	db.rwMutex.Lock()
	defer db.rwMutex.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if db.connection == nil {
		return fmt.Errorf("no active connection with db")
	}

	if metric.MType == "gauge" {
		if metric.Value == nil {
			return storage.ErrInvalidMetricValue
		}
		err := db.connection.AppendGaugeMetric(ctx, metric)
		if err != nil {
			return fmt.Errorf("AppendMetric: %v", err)
		}
	} else if metric.MType == "counter" {
		if metric.Delta == nil {
			return storage.ErrInvalidMetricValue
		}
		err := db.connection.AppendCounterMetric(ctx, metric)
		if err != nil {
			return fmt.Errorf("AppendMetric: %v", err)
		}
	} else {
		return fmt.Errorf("metric type: %s is not supported", metric.MType)
	}
	return nil
}

// GetAllMetrics TODO: error handling
func (db *DBStorage) GetAllMetrics() []models.Metrics {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if db.connection == nil {
		return nil
	}
	metrics, err := db.connection.GetAllMetrics(ctx)
	if err != nil {
		logger.Log.Warn("", zap.Error(err))
		return nil
	}
	return metrics
}

func (db *DBStorage) AppendMetrics(metrics []models.Metrics) error {
	db.rwMutex.Lock()
	defer db.rwMutex.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if db.connection == nil {
		return fmt.Errorf("no active connection with db")
	}

	return db.connection.AppendBatch(ctx, metrics)
}
