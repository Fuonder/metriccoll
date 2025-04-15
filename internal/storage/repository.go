package storage

import (
	"context"
	"time"

	"github.com/Fuonder/metriccoll.git/internal/models"
)

//type Storage interface {
//	GetMetricByName(name string, mType string) (models.Metrics, error)
//	AppendMetric(metric models.Metrics) error
//	AppendMetrics([]models.Metrics) error
//	GetAllMetrics() []models.Metrics
//	DumpMetrics() error
//	loadMetricsFromFile() error
//	CheckConnection() error
//}

type MetricReader interface {
	GetAllMetrics() []models.Metrics
	GetMetricByName(name string, mType string) (models.Metrics, error)
}
type MetricWriter interface {
	AppendMetric(metric models.Metrics) error
	AppendMetrics([]models.Metrics) error
}
type MetricFileHandler interface {
	DumpMetrics() error
	loadMetricsFromFile() error
}
type MetricDatabaseHandler interface {
	CheckConnection() error
}

type Collection interface {
	ReadValues()
	UpdateValues(ctx context.Context, interval time.Duration)
	GetCounterList() map[string]models.Counter
	GetGaugeList() map[string]models.Gauge
}

type DBConnection interface {
	DBReader
	DBWriter
	CreateTablesContext(ctx context.Context) error
	TryConnectContext(ctx context.Context) error
	Close() error
}

type DBReader interface {
	GetGaugeMetric(ctx context.Context, name string) (models.Metrics, error)
	GetCounterMetric(ctx context.Context, name string) (models.Metrics, error)
	GetAllMetrics(ctx context.Context) ([]models.Metrics, error)
}
type DBWriter interface {
	AppendGaugeMetric(ctx context.Context, metric models.Metrics) error
	AppendCounterMetric(ctx context.Context, metric models.Metrics) error
	AppendBatch(ctx context.Context, metrics []models.Metrics) error
}

//type DBReader interface{
//	QueryRowContext(ctx context.Context, query string,   args ...interface{}) (*sql.Row, error)
//	QueryContext(ctx context.Context, query string, args ...interface{})  (*sql.Rows, error)
//}
