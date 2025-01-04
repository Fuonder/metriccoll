package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

var (
	pollInterval   = 2 * time.Second
	reportInterval = 10 * time.Second
	mc, _          = NewMetricsCollection()
)

func updateValues() {
	for {
		log.Println("Updating metrics collection")
		mc.ReadValues()
		time.Sleep(pollInterval)
	}
}

func main() {
	go updateValues()
	for {
		log.Println("Sending metrics collection")
		mc.mu.Lock()
		err := SendMetrics()
		if err != nil {
			return
		}
		mc.mu.Unlock()
		time.Sleep(reportInterval)
	}

}

func SendMetrics() error {
	for name, value := range mc.gMetrics {
		url := "http://localhost:8080/update/"
		url += value.Type() + "/" + name + "/" + strconv.FormatFloat(float64(value), 'f', -1, 64)
		request, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			return fmt.Errorf("error creating request: %v", err)
		}
		request.Header.Add("Content-Type", "text/plain")
		client := &http.Client{}
		resp, err := client.Do(request)
		defer resp.Body.Close()
		if err != nil {
			return fmt.Errorf("could not send request: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("received status code: %d", resp.StatusCode)
		}

	}
	for name, value := range mc.cMetrics {
		url := "http://localhost:8080/update/"
		url += value.Type() + "/" + name + "/" + strconv.FormatInt(int64(value), 10)
		request, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			return fmt.Errorf("error creating request: %v", err)
		}
		request.Header.Add("Content-Type", "text/plain")
		client := &http.Client{}
		resp, err := client.Do(request)
		defer resp.Body.Close()
		if err != nil {
			return fmt.Errorf("could not send request: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("received status code: %d", resp.StatusCode)
		}
	}
	return nil
}
