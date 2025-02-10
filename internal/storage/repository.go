package storage

import (
	"github.com/Fuonder/metriccoll.git/internal/models"
	"time"
)

type Storage interface {
	GetMetricByName(name string, mType string) (models.Metrics, error)
	AppendMetric(metric models.Metrics) error
	GetAllMetrics() []models.Metrics
	DumpMetrics() error
	loadMetricsFromFile() error
	CheckConnection() error
}

type Collection interface {
	ReadValues()
	UpdateValues(interval time.Duration, stopChan chan struct{})
	GetCounterList() map[string]models.Counter
	GetGaugeList() map[string]models.Gauge
}
