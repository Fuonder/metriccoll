package main

import (
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"log"
	"strconv"
	"time"
)

var mc = MetricsCollection{
	gMetrics: make(map[string]gauge),
	cMetrics: map[string]counter{"PollCount": 0},
}

var (
	ErrCouldNotCreateRequest = errors.New("could not create request")
	ErrCouldNotSendRequest   = errors.New("could not send request")
	ErrWrongResponseStatus   = errors.New("wrong request data or metrics value")
)

//func updateValues() {
//	for {
//		time.Sleep(pollInterval)
//		log.Println("Updating metrics collection")
//		mc.ReadValues()
//	}
//}

func main() {
	err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}
	s := time.Now()
	go func() {
		for {
			mc.ReadValues()
			fmt.Printf("read time: %s\n", time.Now().Sub(s).String())
			time.Sleep(opt.pollInterval)
		}
	}()
	for {
		time.Sleep(opt.reportInterval)
		fmt.Println(time.Now().Sub(s))
		_ = SendMetrics()
		fmt.Println(time.Now().Sub(s))
	}

}

//func SendMetrics() error {
//	client := resty.New()
//	errChan := make(chan error, len(mc.gMetrics)+len(mc.cMetrics))
//	var wg sync.WaitGroup
//
//	// Helper function to send a metric
//	sendMetric := func(url string) {
//		defer wg.Done()
//		resp, err := client.R().
//			SetHeader("Content-Type", "text/plain").
//			Post(url)
//
//		if err != nil {
//			errChan <- fmt.Errorf("%w: %s", ErrCouldNotSendRequest, err)
//			return
//		}
//
//		if resp.StatusCode() != 200 {
//			errChan <- ErrWrongResponseStatus
//		}
//	}
//
//	// Sending metrics for gMetrics in parallel
//	for name, value := range mc.gMetrics {
//		url := "http://localhost:8080/update/" + value.Type() + "/" + name + "/" + strconv.FormatFloat(float64(value), 'f', -1, 64)
//		wg.Add(1)
//		go sendMetric(url)
//	}
//
//	// Sending metrics for cMetrics in parallel
//	for name, value := range mc.cMetrics {
//		url := "http://localhost:8080/update/" + value.Type() + "/" + name + "/" + strconv.FormatInt(int64(value), 10)
//		wg.Add(1)
//		go sendMetric(url)
//	}
//
//	// Wait for all goroutines to finish
//	wg.Wait()
//
//	// Close the error channel to check if any error occurred
//	close(errChan)
//
//	// Check for errors
//	for err := range errChan {
//		if err != nil {
//			return err
//		}
//	}
//
//	return nil
//}

func SendMetrics() error {
	client := resty.New()

	// Sending metrics for gMetrics
	for name, value := range mc.gMetrics {
		url := "http://localhost:8080/update/" + value.Type() + "/" + name + "/" + strconv.FormatFloat(float64(value), 'f', -1, 64)
		resp, err := client.R().
			SetHeader("Content-Type", "text/plain").
			Post(url)

		if err != nil {
			return fmt.Errorf("%w: %s", ErrCouldNotSendRequest, err)
		}

		if resp.StatusCode() != 200 {
			return ErrWrongResponseStatus
		}
	}

	// Sending metrics for cMetrics
	for name, value := range mc.cMetrics {
		url := "http://localhost:8080/update/" + value.Type() + "/" + name + "/" + strconv.FormatInt(int64(value), 10)
		resp, err := client.R().
			SetHeader("Content-Type", "text/plain").
			Post(url)

		if err != nil {
			return fmt.Errorf("%w: %s", ErrCouldNotSendRequest, err)
		}

		if resp.StatusCode() != 200 {
			return ErrWrongResponseStatus
		}
	}

	return nil
}

//	func SendMetrics() error {
//		var resp *http.Response
//		//log.Println("Sending metrics collection")
//		for name, value := range mc.gMetrics {
//			url := "http://localhost:8080/update/"
//			url += value.Type() + "/" + name + "/" + strconv.FormatFloat(float64(value), 'f', -1, 64)
//			request, err := http.NewRequest(http.MethodPost, url, nil)
//			if err != nil {
//				return ErrCouldNotCreateRequest
//			}
//			request.Header.Add("Content-Type", "text/plain")
//			client := &http.Client{}
//			resp, err = client.Do(request)
//			if err != nil {
//				return fmt.Errorf("%w: %s", ErrCouldNotSendRequest, err)
//			}
//			defer resp.Body.Close()
//			if resp.StatusCode != http.StatusOK {
//				return ErrWrongResponseStatus
//			}
//		}
//		for name, value := range mc.cMetrics {
//			url := "http://localhost:8080/update/"
//			url += value.Type() + "/" + name + "/" + strconv.FormatInt(int64(value), 10)
//			request, err := http.NewRequest(http.MethodPost, url, nil)
//			if err != nil {
//				return ErrCouldNotCreateRequest
//			}
//			request.Header.Add("Content-Type", "text/plain")
//			client := &http.Client{}
//			resp, err = client.Do(request)
//			if err != nil {
//				return ErrCouldNotSendRequest
//			}
//			defer resp.Body.Close()
//			if resp.StatusCode != http.StatusOK {
//				return ErrWrongResponseStatus
//			}
//		}
//		return nil
//	}
