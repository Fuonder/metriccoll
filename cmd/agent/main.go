package main

import (
	"errors"
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/storage"
	"github.com/go-resty/resty/v2"
	"log"
	"strconv"
	"time"
)

var (
	ErrCouldNotSendRequest = errors.New("could not send request")
	ErrWrongResponseStatus = errors.New("wrong request data or metrics value")
)

func main() {
	mc, err := storage.NewMetricsCollection()
	if err != nil {
		log.Fatal(err)
	}

	err = parseFlags()
	if err != nil {
		log.Fatal(err)
	}

	ch := make(chan struct{})
	mc.UpdateValues(CliOpt.PollInterval, ch)

	for {
		time.Sleep(CliOpt.ReportInterval)
		err = SendMetrics(mc)
		if err != nil {
			close(ch)
			time.Sleep(2 * time.Second)
			log.Fatal(err)
		}
		//err = testAll()
		//if err != nil {
		//	close(ch)
		//	time.Sleep(2 * time.Second)
		//	log.Fatal(err)
		//}
	}
}

//func testAll() error {
//	client := resty.New()
//	resp, err := client.R().SetHeader("Content-Type", "text/plain").Get("http://localhost:8080/")
//	if err != nil {
//		return err
//	}
//	fmt.Println(resp)
//	return nil
//}

func SendMetrics(mc storage.Collection) error {
	client := resty.New()
	gMetrics := mc.GetGaugeList()
	cMetrics := mc.GetCounterList()

	for name, value := range gMetrics {
		url := "http://" + CliOpt.NetAddr.String() + "/update/" + value.Type() +
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
	for name, value := range cMetrics {
		url := "http://" + CliOpt.NetAddr.String() + "/update/" + value.Type() +
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
