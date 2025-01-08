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
	ErrCouldNotSendRequest = errors.New("could not send request")
	ErrWrongResponseStatus = errors.New("wrong request data or metrics value")
)

func main() {
	err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}

	mc.UpdateValues(opt.pollInterval)
	for {
		time.Sleep(opt.reportInterval)
		_ = SendMetrics()
	}
}

func SendMetrics() error {
	client := resty.New()
	for name, value := range mc.gMetrics {
		url := "http://" + opt.netAddr.String() + "/update/" + value.Type() +
			"/" + name + "/" + strconv.FormatFloat(float64(value), 'f', -1, 64)
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
	for name, value := range mc.cMetrics {
		url := "http://" + opt.netAddr.String() + "/update/" + value.Type() +
			"/" + name + "/" + strconv.FormatInt(int64(value), 10)
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
