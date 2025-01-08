package main

import (
	"errors"
	"log"
	"math/rand"
	"runtime"
	"time"
)

var ErrFieldNotFound = errors.New("field not found")

type gauge float64

func (t gauge) Type() string {
	return "gauge"
}

type counter int64

func (t counter) Type() string {
	return "counter"
}

type MetricsCollection struct {
	gMetrics map[string]gauge
	cMetrics map[string]counter
	//mu       sync.Mutex
}

func NewMetricsCollection() (*MetricsCollection, error) {
	mc := MetricsCollection{
		gMetrics: make(map[string]gauge),
		cMetrics: map[string]counter{"PollCount": 0},
	}
	return &mc, nil
}

func (mc *MetricsCollection) ReadValues() {
	// mc.mu.Lock()
	// defer mc.mu.Unlock()

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	mc.cMetrics["PollCount"]++

	mc.gMetrics = map[string]gauge{
		"Alloc":         gauge(ms.Alloc),
		"BuckHashSys":   gauge(ms.BuckHashSys),
		"Frees":         gauge(ms.Frees),
		"GCCPUFraction": gauge(ms.GCCPUFraction),
		"GCSys":         gauge(ms.GCSys),
		"HeapAlloc":     gauge(ms.HeapAlloc),
		"HeapIdle":      gauge(ms.HeapIdle),
		"HeapInuse":     gauge(ms.HeapInuse),
		"HeapObjects":   gauge(ms.HeapObjects),
		"HeapReleased":  gauge(ms.HeapReleased),
		"HeapSys":       gauge(ms.HeapSys),
		"LastGC":        gauge(ms.LastGC),
		"Lookups":       gauge(ms.Lookups),
		"MCacheInuse":   gauge(ms.MCacheInuse),
		"MCacheSys":     gauge(ms.MCacheSys),
		"MSpanInuse":    gauge(ms.MSpanInuse),
		"MSpanSys":      gauge(ms.MSpanSys),
		"Mallocs":       gauge(ms.Mallocs),
		"NextGC":        gauge(ms.NextGC),
		"NumForcedGC":   gauge(ms.NumForcedGC),
		"NumGC":         gauge(ms.NumGC),
		"OtherSys":      gauge(ms.OtherSys),
		"PauseTotalNs":  gauge(ms.PauseTotalNs),
		"StackInuse":    gauge(ms.StackInuse),
		"StackSys":      gauge(ms.StackSys),
		"Sys":           gauge(ms.Sys),
		"TotalAlloc":    gauge(ms.TotalAlloc),
		"RandomValue":   gauge(rand.Float64() * 100),
	}
	// log.Printf("PollCount:\t%d\n", mc.cMetrics["PollCount"])
}

func (mc *MetricsCollection) UpdateValues(interval time.Duration) {
	go func() {
		for {
			log.Println("Updating metrics collection")
			mc.ReadValues()
			time.Sleep(interval)
		}
	}()
}

func (mc *MetricsCollection) getPollCount() (counter, error) {
	//mc.mu.Lock()
	//defer mc.mu.Unlock()
	if _, ok := mc.cMetrics["PollCount"]; ok {
		return mc.cMetrics["PollCount"], nil
	}
	return 0, ErrFieldNotFound

}
