package main

import "sync"

type gauge float64
type counter int64
type MemStorage struct {
	gMetric map[string]gauge
	cMetric map[string]counter
	mu      sync.Mutex
}

func NewMemStorage() (*MemStorage, error) {
	ms := MemStorage{
		gMetric: make(map[string]gauge),
		cMetric: make(map[string]counter),
	}
	return &ms, nil
}

func (ms *MemStorage) AppendGaugeMetric(name string, value gauge) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.gMetric[name] = value
}
func (ms *MemStorage) AppendCounterMetric(name string, value counter) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if metricValue, exists := ms.cMetric[name]; exists {
		ms.cMetric[name] = metricValue + value
	} else {
		ms.cMetric[name] = value
	}
}
