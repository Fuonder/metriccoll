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
	"net/http"
	"strconv"
	"time"
)

var (
	ErrCouldNotSendRequest = errors.New("could not send request")
	ErrWrongResponseStatus = errors.New("wrong request data or metrics value")
)

func CheckServerConnection(url string) error {
	// Устанавливаем таймаут для запроса
	client := http.Client{
		Timeout: 5 * time.Second, // Таймаут 5 секунд
	}

	// Отправляем GET запрос на сервер
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем код ответа
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned non-OK status: %d", resp.StatusCode)
	}

	// Если соединение успешное, возвращаем nil (ошибки нет)
	return nil
}

func main() {
	fmt.Println("Starting agent")
	mc, err := storage.NewMetricsCollection()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("metric collection creation success")

	err = parseFlags()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("parse flags success")

	err = CheckServerConnection("http://" + CliOpt.NetAddr.String())
	if err != nil {
		fmt.Println("Connection check failed:", err)
	} else {
		fmt.Println("Server is reachable!")
	}

	ch := make(chan struct{})
	mc.UpdateValues(CliOpt.PollInterval, ch)

	for {
		time.Sleep(CliOpt.ReportInterval)
		fmt.Println("sending metrics")
		err = SendMetricsJSON(mc)
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

var globalcounter = 0

func SendMetricsJSON(mc storage.Collection) error {
	client := resty.New()
	gMetrics := mc.GetGaugeList()
	cMetrics := mc.GetCounterList()
	for name, value := range cMetrics {
		mt := models.Metrics{
			ID:    name,
			MType: "counter",
			Delta: (*int64)(&value),
			Value: nil,
		}

		globalcounter++
		//out, err := json.Marshal(mt)
		//if err != nil {
		//	return fmt.Errorf("json marshal: %v", err)
		//}
		fmt.Println("-----------------------------------------------------sending")
		fmt.Println(mt)
		fmt.Println(globalcounter)
		url := "http://" + CliOpt.NetAddr.String() + "/update"
		//body, err := json.Marshal(mt)
		//if err != nil {
		//	return fmt.Errorf("failed to marshal request body: %w", err)
		//}
		//
		//// Create the request
		//req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
		//if err != nil {
		//	return fmt.Errorf("failed to create request: %w", err)
		//}
		//
		//// Set the headers
		//req.Header.Set("Content-Type", "application/json")
		//
		//// Send the request
		//client := &http.Client{}
		//resp, err := client.Do(req)
		//if err != nil {
		//	return fmt.Errorf("could not send request: %w", err)
		//}
		//defer resp.Body.Close()
		//
		//// Read the response body for debugging purposes
		//respBody, _ := io.ReadAll(resp.Body)
		//
		//// Log the response status and body for troubleshooting
		//fmt.Printf("Response status: %d\n", resp.StatusCode)
		//fmt.Printf("Response body: %s\n", respBody)
		//
		//// Check the response status
		//if resp.StatusCode != 200 {
		//	return fmt.Errorf("unexpected response status: %d", resp.StatusCode)
		//}
		//
		//return nil
		body, err := json.Marshal(mt)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(body).
			Post(url)
		if err != nil {
			return fmt.Errorf("1%w: %s", ErrCouldNotSendRequest, err)
		}
		if resp.StatusCode() != 200 {
			return ErrWrongResponseStatus
		}
	}
	for name, value := range gMetrics {
		globalcounter++
		var mt, res models.Metrics
		mt.ID = name
		mt.MType = "gauge"
		mt.Value = (*float64)(&value)
		//out, err := json.Marshal(mt)
		//if err != nil {
		//	return fmt.Errorf("json marshal: %v", err)
		//}
		fmt.Println("sending")
		fmt.Println(mt)
		fmt.Println(globalcounter)
		url := "http://" + CliOpt.NetAddr.String() + "/update"
		cli := client.R()
		cli.SetHeader("Content-Type", "application/json")
		//cli.SetHeader("Accept", "application/json")
		cli.SetBody(&mt)
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

	return nil
}
