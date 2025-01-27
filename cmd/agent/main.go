package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/logger"
	"github.com/Fuonder/metriccoll.git/internal/models"
	"github.com/Fuonder/metriccoll.git/internal/storage"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
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
		logger.Log.Info("sending metric", zap.String("url", url))
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
		logger.Log.Info("sending metric", zap.String("url", url))
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

func SendMetricsJSON(mc storage.Collection) error {
	client := resty.New()
	gMetrics := mc.GetGaugeList()
	cMetrics := mc.GetCounterList()

	for name, value := range gMetrics {
		var mt, res models.Metrics
		mt.ID = name
		mt.MType = "gauge"
		mt.Value = (*float64)(&value)
		out, err := json.Marshal(mt)
		if err != nil {
			return fmt.Errorf("json marshal: %v", err)
		}
		fmt.Println("sending")
		fmt.Println(mt)
		url := "http://" + CliOpt.NetAddr.String() + "/update/"
		cli := client.R()
		cli.SetHeader("Content-Type", "application/json")
		cli.SetHeader("Accept", "application/json")
		cli.SetBody(out)
		cli.SetResult(res)
		resp, err := cli.Post(url)
		if err != nil {
			fmt.Printf("Errors while sending metrics json: %v\n", err)
			fmt.Printf("URL: %s\n", url)
			fmt.Printf("respnse status: %d\n", resp.StatusCode())
			fmt.Printf("value %f\n", value)
			fmt.Printf("response body:\n %s", string(resp.Body()))
			fmt.Printf("result:\n")
			fmt.Println(res)
			return fmt.Errorf("2%w: %s", ErrCouldNotSendRequest, err)
		}
		if resp.StatusCode() != 200 {
			fmt.Println("GOT NOT 200 RESPONSE")
			fmt.Printf("response body:\n %s", string(resp.Body()))
			return ErrWrongResponseStatus
		}
	}
	for name, value := range cMetrics {
		var mt models.Metrics
		mt.ID = name
		mt.MType = "counter"
		mt.Delta = (*int64)(&value)
		out, err := json.Marshal(mt)
		if err != nil {
			return fmt.Errorf("json marshal: %v", err)
		}

		url := "http://" + CliOpt.NetAddr.String() + "/update/"
		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(out).
			Post(url)
		if err != nil {
			return fmt.Errorf("1%w: %s", ErrCouldNotSendRequest, err)
		}
		if resp.StatusCode() != 200 {
			return ErrWrongResponseStatus
		}
	}
	return nil
}
