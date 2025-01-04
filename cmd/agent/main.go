package main

import (
	"errors"
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
	mc.UpdateValues(pollInterval)
	for {
		time.Sleep(reportInterval)
		err := SendMetrics()
		if err != nil {
			log.Fatal(err)
		}
	}

}

func SendMetrics() error {
	var resp *http.Response
	log.Println("Sending metrics collection")
	for name, value := range mc.gMetrics {
		url := "http://localhost:8080/update/"
		url += value.Type() + "/" + name + "/" + strconv.FormatFloat(float64(value), 'f', -1, 64)
		request, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			return ErrCouldNotCreateRequest
		}
		request.Header.Add("Content-Type", "text/plain")
		client := &http.Client{}
		resp, err = client.Do(request)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrCouldNotSendRequest, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return ErrWrongResponseStatus
		}
	}
	for name, value := range mc.cMetrics {
		url := "http://localhost:8080/update/"
		url += value.Type() + "/" + name + "/" + strconv.FormatInt(int64(value), 10)
		request, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			return ErrCouldNotCreateRequest
		}
		request.Header.Add("Content-Type", "text/plain")
		client := &http.Client{}
		resp, err = client.Do(request)
		if err != nil {
			return ErrCouldNotSendRequest
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return ErrWrongResponseStatus
		}
	}
	return nil
}
