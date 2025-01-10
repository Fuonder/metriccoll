package storage

import (
	"fmt"
	model "github.com/Fuonder/metriccoll.git/internal/models"
	"sync"
)

type memStorage struct {
	gMetric map[string]model.Gauge
	cMetric map[string]model.Counter
	mu      sync.Mutex
}

func NewMemStorage() (*memStorage, error) {
	ms := memStorage{
		gMetric: make(map[string]model.Gauge),
		cMetric: make(map[string]model.Counter),
	}
	return &ms, nil
}

func (ms *memStorage) AppendGaugeMetric(name string, value model.Gauge) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.gMetric[name] = value
}
func (ms *memStorage) AppendCounterMetric(name string, value model.Counter) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if metricValue, exists := ms.cMetric[name]; exists {
		ms.cMetric[name] = metricValue + value
	} else {
		ms.cMetric[name] = value
	}
}

func (ms *memStorage) GetGaugeMetric(name string) (model.Gauge, error) {
	metric, ok := ms.gMetric[name]
	if !ok {
		return model.Gauge(0), fmt.Errorf("%w: %s", ErrMetricNotFound, name)
	}
	return metric, nil
}

func (ms *memStorage) GetGaugeList() map[string]model.Gauge {
	return ms.gMetric
}

func (ms *memStorage) GetCounterMetric(name string) (model.Counter, error) {
	metric, ok := ms.cMetric[name]
	if !ok {
		return model.Counter(0), fmt.Errorf("%w: %s", ErrMetricNotFound, name)
	}
	return metric, nil
}

func (ms *memStorage) GetCounterList() map[string]model.Counter {
	return ms.cMetric
}
