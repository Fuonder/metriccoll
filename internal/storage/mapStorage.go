package storage

import (
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/models"
	"sync"
)

// Deprecated: Memory storage does not supported from 0.1.9 version
type memStorage struct {
	gMetric map[string]models.Gauge
	cMetric map[string]models.Counter
	mu      sync.Mutex
}

func NewMemStorage() (*memStorage, error) {
	ms := memStorage{
		gMetric: make(map[string]models.Gauge),
		cMetric: make(map[string]models.Counter),
	}
	return &ms, nil
}

func (ms *memStorage) loadMetricsFromFile() error {
	return fmt.Errorf("loading from file is not yet implemented" +
		"consider using \"jsonStorage\"")
}

func (ms *memStorage) DumpMetrics() error {
	return fmt.Errorf("dump to file is not yet implemented, " +
		"consider using \"jsonStorage\"")
}

func (ms *memStorage) AppendMetric(metric models.Metrics) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if metric.MType == "gauge" {
		if metric.Value == nil {
			return ErrInvalidMetricValue
		}
		ms.gMetric[metric.ID] = models.Gauge(*metric.Value)
		return nil
	} else if metric.MType == "counter" {
		if metric.Delta == nil {
			return ErrInvalidMetricValue
		}
		if metricValue, exists := ms.cMetric[metric.ID]; exists {
			ms.cMetric[metric.ID] = metricValue + models.Counter(*metric.Delta)
			return nil
		} else {
			ms.cMetric[metric.ID] = models.Counter(*metric.Delta)
			return nil
		}
	} else {
		return fmt.Errorf("unknown metric type: %s", metric.MType)
	}
}

func (ms *memStorage) GetMetricByName(name string, mType string) (models.Metrics, error) {
	if mType == "gauge" {
		metricValGauge, err := ms.getGaugeMetric(name)
		if err != nil {
			return models.Metrics{}, err
		}
		return models.Metrics{ID: name, MType: "gauge", Value: (*float64)(&metricValGauge)}, nil
	} else if mType == "counter" {
		metricValCounter, err := ms.getCounterMetric(name)
		if err != nil {
			return models.Metrics{}, err
		}
		return models.Metrics{ID: name, MType: "counter", Delta: (*int64)(&metricValCounter)}, nil
	} else {
		return models.Metrics{}, fmt.Errorf("unknown metric type: %s", mType)
	}
}

func (ms *memStorage) GetAllMetrics() []models.Metrics {
	gMetrics := ms.getGaugeList()
	cMetrics := ms.getCounterList()
	md := make([]models.Metrics, len(gMetrics)+len(cMetrics))
	for name, value := range gMetrics {
		md = append(md, models.Metrics{ID: name, MType: "gauge", Value: (*float64)(&value)})
	}
	for name, value := range cMetrics {
		md = append(md, models.Metrics{ID: name, MType: "counter", Delta: (*int64)(&value)})
	}
	return md
}

func (ms *memStorage) appendGaugeMetric(name string, value models.Gauge) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.gMetric[name] = value
}
func (ms *memStorage) appendCounterMetric(name string, value models.Counter) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if metricValue, exists := ms.cMetric[name]; exists {
		ms.cMetric[name] = metricValue + value
	} else {
		ms.cMetric[name] = value
	}
}

func (ms *memStorage) getGaugeMetric(name string) (models.Gauge, error) {
	metric, ok := ms.gMetric[name]
	if !ok {
		return models.Gauge(0), fmt.Errorf("%w: %s", ErrMetricNotFound, name)
	}
	return metric, nil
}

func (ms *memStorage) getGaugeList() map[string]models.Gauge {
	return ms.gMetric
}

func (ms *memStorage) getCounterMetric(name string) (models.Counter, error) {
	metric, ok := ms.cMetric[name]
	if !ok {
		return models.Counter(0), fmt.Errorf("%w: %s", ErrMetricNotFound, name)
	}
	return metric, nil
}

func (ms *memStorage) getCounterList() map[string]models.Counter {
	return ms.cMetric
}

func (ms *memStorage) CheckConnection() error {
	return fmt.Errorf("not implemented")
}
