package main

import (
	"errors"
	"fmt"
	"sync"
)

type gauge float64
type counter int64

var ErrMetricNotFound = errors.New("metric with such key is not found")

type MemStorage interface {
	NewMemStorage() MemStorage
	AppendGaugeMetric()
	AppendCounterMetric()
	GetGaugeMetric(key string) (gauge, error)
	GetCounterMetric(key string) (counter, error)
}

type Storage struct {
	gMetric map[string]gauge
	cMetric map[string]counter
	mu      sync.Mutex
}

func NewMemStorage() (*Storage, error) {
	ms := Storage{
		gMetric: make(map[string]gauge),
		cMetric: make(map[string]counter),
	}
	return &ms, nil
}

func (ms *Storage) AppendGaugeMetric(name string, value gauge) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.gMetric[name] = value
}
func (ms *Storage) AppendCounterMetric(name string, value counter) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if metricValue, exists := ms.cMetric[name]; exists {
		ms.cMetric[name] = metricValue + value
	} else {
		ms.cMetric[name] = value
	}
}

func (ms *Storage) GetGaugeMetric(name string) (gauge, error) {
	if metric, ok := ms.gMetric[name]; ok {
		return metric, nil
	} else {
		return gauge(0), fmt.Errorf("%w: %s", ErrMetricNotFound, name)
	}
}

func (ms *Storage) GetCounterMetric(name string) (counter, error) {
	if metric, ok := ms.cMetric[name]; ok {
		return metric, nil
	} else {
		return counter(0), fmt.Errorf("%w: %s", ErrMetricNotFound, name)
	}
}
