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
	fmt.Printf("opt:\n\t%s\n\t%s\n", opt.pollInterval.String(), opt.reportInterval.String())
	//s := time.Now()
	go func() {
		for {
			mc.UpdateValues(opt.pollInterval)
			//fmt.Printf("read time: %s\n", time.Since(s).String())
			//time.Sleep(opt.pollInterval)
		}
	}()
	for {
		time.Sleep(opt.reportInterval)
		_ = SendMetrics()
		//fmt.Println(time.Since(s))
	}

}

func SendMetrics() error {
	client := resty.New()

	for name, value := range mc.gMetrics {
		url := "http://" + opt.netAddr.String() + "/update/" + value.Type() + "/" + name + "/" + strconv.FormatFloat(float64(value), 'f', -1, 64)
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
		url := "http://" + opt.netAddr.String() + "/update/" + value.Type() + "/" + name + "/" + strconv.FormatInt(int64(value), 10)
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
