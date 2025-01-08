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
	//mc.mu.Lock()
	//defer mc.mu.Unlock()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	rNum := rand.Float64() * 100

	mc.cMetrics["PollCount"] = mc.cMetrics["PollCount"] + 1

	mc.gMetrics["Alloc"] = gauge(ms.Alloc)
	mc.gMetrics["BuckHashSys"] = gauge(ms.BuckHashSys)
	mc.gMetrics["Frees"] = gauge(ms.Frees)
	mc.gMetrics["GCCPUFraction"] = gauge(ms.GCCPUFraction)
	mc.gMetrics["GCSys"] = gauge(ms.GCSys)
	mc.gMetrics["HeapAlloc"] = gauge(ms.HeapAlloc)
	mc.gMetrics["HeapIdle"] = gauge(ms.HeapIdle)
	mc.gMetrics["HeapInuse"] = gauge(ms.HeapInuse)
	mc.gMetrics["HeapObjects"] = gauge(ms.HeapObjects)
	mc.gMetrics["HeapReleased"] = gauge(ms.HeapReleased)
	mc.gMetrics["HeapSys"] = gauge(ms.HeapSys)
	mc.gMetrics["LastGC"] = gauge(ms.LastGC)
	mc.gMetrics["Lookups"] = gauge(ms.Lookups)
	mc.gMetrics["MCacheInuse"] = gauge(ms.MCacheInuse)
	mc.gMetrics["MCacheSys"] = gauge(ms.MCacheSys)
	mc.gMetrics["MSpanInuse"] = gauge(ms.MSpanInuse)
	mc.gMetrics["MSpanSys"] = gauge(ms.MSpanSys)
	mc.gMetrics["Mallocs"] = gauge(ms.Mallocs)
	mc.gMetrics["NextGC"] = gauge(ms.NextGC)
	mc.gMetrics["NumForcedGC"] = gauge(ms.NumForcedGC)
	mc.gMetrics["NumGC"] = gauge(ms.NumGC)
	mc.gMetrics["OtherSys"] = gauge(ms.OtherSys)
	mc.gMetrics["PauseTotalNs"] = gauge(ms.PauseTotalNs)
	mc.gMetrics["StackInuse"] = gauge(ms.StackInuse)
	mc.gMetrics["StackSys"] = gauge(ms.StackSys)
	mc.gMetrics["Sys"] = gauge(ms.Sys)
	mc.gMetrics["TotalAlloc"] = gauge(ms.TotalAlloc)
	mc.gMetrics["RandomValue"] = gauge(rNum)
	//log.Printf("PollCount:\t%d\n", mc.cMetrics["PollCount"])

}

func (mc *MetricsCollection) UpdateValues(interval time.Duration) {
	go func() {
		for {

			time.Sleep(interval)
			log.Println("Updating metrics collection")
			mc.ReadValues()
			//time.Sleep(interval)
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
