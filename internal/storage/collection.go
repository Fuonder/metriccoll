package storage

import (
	"context"
	"github.com/Fuonder/metriccoll.git/internal/logger"
	model "github.com/Fuonder/metriccoll.git/internal/models"
	"math/rand/v2"
	"runtime"
	"sync"
	"time"
)

type MetricsCollection struct {
	gMetrics map[string]model.Gauge
	cMetrics map[string]model.Counter
	mu       sync.Mutex
}

func NewMetricsCollection() (*MetricsCollection, error) {
	mc := MetricsCollection{
		gMetrics: make(map[string]model.Gauge),
		cMetrics: map[string]model.Counter{"PollCount": 0},
		mu:       sync.Mutex{},
	}
	return &mc, nil
}

func (mc *MetricsCollection) ReadValues() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	mc.cMetrics["PollCount"]++

	mc.gMetrics = map[string]model.Gauge{
		"Alloc":         model.Gauge(ms.Alloc),
		"BuckHashSys":   model.Gauge(ms.BuckHashSys),
		"Frees":         model.Gauge(ms.Frees),
		"GCCPUFraction": model.Gauge(ms.GCCPUFraction),
		"GCSys":         model.Gauge(ms.GCSys),
		"HeapAlloc":     model.Gauge(ms.HeapAlloc),
		"HeapIdle":      model.Gauge(ms.HeapIdle),
		"HeapInuse":     model.Gauge(ms.HeapInuse),
		"HeapObjects":   model.Gauge(ms.HeapObjects),
		"HeapReleased":  model.Gauge(ms.HeapReleased),
		"HeapSys":       model.Gauge(ms.HeapSys),
		"LastGC":        model.Gauge(ms.LastGC),
		"Lookups":       model.Gauge(ms.Lookups),
		"MCacheInuse":   model.Gauge(ms.MCacheInuse),
		"MCacheSys":     model.Gauge(ms.MCacheSys),
		"MSpanInuse":    model.Gauge(ms.MSpanInuse),
		"MSpanSys":      model.Gauge(ms.MSpanSys),
		"Mallocs":       model.Gauge(ms.Mallocs),
		"NextGC":        model.Gauge(ms.NextGC),
		"NumForcedGC":   model.Gauge(ms.NumForcedGC),
		"NumGC":         model.Gauge(ms.NumGC),
		"OtherSys":      model.Gauge(ms.OtherSys),
		"PauseTotalNs":  model.Gauge(ms.PauseTotalNs),
		"StackInuse":    model.Gauge(ms.StackInuse),
		"StackSys":      model.Gauge(ms.StackSys),
		"Sys":           model.Gauge(ms.Sys),
		"TotalAlloc":    model.Gauge(ms.TotalAlloc),
		"RandomValue":   model.Gauge(rand.Float64() * 100),
	}
	// log.Printf("PollCount:\t%d\n", mc.cMetrics["PollCount"])
}

func (mc *MetricsCollection) UpdateValues(ctx context.Context, interval time.Duration) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Log.Debug("Stopping metrics collection")
				return
			default:
				logger.Log.Info("Updating metrics collection")
				mc.ReadValues()
				time.Sleep(interval)
			}
		}
	}()
}

func (mc *MetricsCollection) GetCounterList() map[string]model.Counter {
	return mc.cMetrics
}

func (mc *MetricsCollection) GetGaugeList() map[string]model.Gauge {
	return mc.gMetrics
}

func (mc *MetricsCollection) GetPollCount() (model.Counter, error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	if _, ok := mc.cMetrics["PollCount"]; ok {
		return mc.cMetrics["PollCount"], nil
	}
	return 0, ErrFieldNotFound

}
