package storage

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/logger"
	"github.com/Fuonder/metriccoll.git/internal/models"
	_ "github.com/jackc/pgx/v5/stdlib"
	"sync"
	"time"
)

type DBFileStoreInfo struct {
	Sync          bool
	StoreInterval time.Duration
	fStore        bool
	fStoragePath  string
}

type DatabaseSettings struct {
	Host     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func NewDatabaseSettings(host string, user string, password string,
	DBName string, SSLMode string) *DatabaseSettings {
	return &DatabaseSettings{host, user, password, DBName, SSLMode}
}

func (s *DatabaseSettings) String() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=%s",
		s.Host, s.User, s.Password, s.DBName, s.SSLMode)
}

// Database TODO: refactor server to use database as storage
type Database struct {
	connection    *sql.DB
	settings      *DatabaseSettings
	metrics       interface{} // metrics TODO: consider type and necessity
	fileStoreInfo DBFileStoreInfo
	fileMu        sync.RWMutex // metrics TODO: consider necessity
	rwMutex       sync.RWMutex
}

func NewDatabase(settings *DatabaseSettings) (*Database, error) {
	var err error
	logger.Log.Info("Connecting to database")
	db := Database{settings: settings}
	db.connection, err = sql.Open("pgx", db.settings.String())
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
	return models.Metrics{}, nil
}

func (db *Database) AppendMetric(metric models.Metrics) error {
	return nil
}
func (db *Database) GetAllMetrics() []models.Metrics {
	return nil
}
func (db *Database) DumpMetrics() error {
	return nil
}

func (db *Database) loadMetricsFromFile() error {
	return nil
}
