package storage

import (
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/models"
	"sync"
)

type JSONStorage struct {
	metrics []models.Metrics
	mu      sync.Mutex
}

func NewJSONStorage() (*JSONStorage, error) {
	return &JSONStorage{metrics: make([]models.Metrics, 0)}, nil
}

func (st *JSONStorage) AppendMetric(metric models.Metrics) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	for i, existingItem := range st.metrics {
		if existingItem.ID == metric.ID {
			if metric.MType == "gauge" {
				st.metrics[i] = metric
				return nil
			} else if metric.MType == "counter" {
				*st.metrics[i].Delta += *metric.Delta
				return nil
			} else {
				return fmt.Errorf("metric type: %s is not supported", metric.MType)
			}
		}
	}
	st.metrics = append(st.metrics, metric)
	return nil
}

func (st *JSONStorage) GetMetricByName(name string) (models.Metrics, error) {
	for i, existingItem := range st.metrics {
		if existingItem.ID == name {
			return st.metrics[i], nil
		}
	}
	return models.Metrics{}, fmt.Errorf("%v: %s", ErrMetricNotFound, name)
}

func (st *JSONStorage) GetAllMetrics() []models.Metrics {
	return st.metrics
}
