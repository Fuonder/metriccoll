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
		if existingItem.ID == metric.ID && existingItem.MType == metric.MType {
			if metric.MType == "gauge" {
				if metric.Value == nil {
					return ErrInvalidMetricValue
				}
				*st.metrics[i].Value = *metric.Value
				fmt.Println(st.GetAllMetrics())
				return nil
			} else if metric.MType == "counter" {
				if metric.Delta == nil {
					return ErrInvalidMetricValue
				}
				*st.metrics[i].Delta += *metric.Delta
				fmt.Println(st.GetAllMetrics())
				return nil
			} else {
				return fmt.Errorf("metric type: %s is not supported", metric.MType)
			}
		}
	}
	if metric.MType == "gauge" {
		if metric.Value == nil {
			return ErrInvalidMetricValue
		}
		st.metrics = append(st.metrics, metric)
		fmt.Println(st.GetAllMetrics())
		return nil
	} else if metric.MType == "counter" {
		if metric.Delta == nil {
			return ErrInvalidMetricValue
		}
		st.metrics = append(st.metrics, metric)
		fmt.Println(st.GetAllMetrics())
		return nil
	} else {
		return fmt.Errorf("metric type: %s is not supported", metric.MType)
	}
}

func (st *JSONStorage) GetMetricByName(name string, mType string) (models.Metrics, error) {
	for i, existingItem := range st.metrics {
		if existingItem.ID == name && existingItem.MType == mType {
			return st.metrics[i], nil
		}
	}
	return models.Metrics{}, fmt.Errorf("%v: %s", ErrMetricNotFound, name)
}

func (st *JSONStorage) GetAllMetrics() []models.Metrics {
	return st.metrics
}
