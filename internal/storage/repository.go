package storage

import (
	model "github.com/Fuonder/metriccoll.git/internal/models"
	"time"
)

type Storage interface {
	AppendGaugeMetric(name string, value model.Gauge)
	AppendCounterMetric(name string, value model.Counter)
	GetGaugeMetric(key string) (model.Gauge, error)
	GetGaugeList() map[string]model.Gauge
	GetCounterMetric(key string) (model.Counter, error)
	GetCounterList() map[string]model.Counter
}

type Collection interface {
	ReadValues()
	UpdateValues(interval time.Duration, stopChan chan struct{})
	GetCounterList() map[string]model.Counter
	GetGaugeList() map[string]model.Gauge
}
