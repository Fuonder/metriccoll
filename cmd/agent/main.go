package main

import (
	"bytes"
	"compress/gzip"
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

type senderFunc func(storage.Collection) error

func retriableHttpSend(sender senderFunc, st storage.Collection) error {
	var err error
	timeouts := []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		logger.Log.Info("sending metrics")
		err = sender(st)
		if err == nil {
			return nil
		}
		if i < len(timeouts) {
			logger.Log.Info("sending metrics failed", zap.Error(err))
			logger.Log.Info("retrying after timeout",
				zap.Duration("timeout", timeouts[i]),
				zap.Int("retry-count", i+1))
			time.Sleep(timeouts[i])
		}
	}
	return err
}

func checkServerConnection(url string) error {
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
	if err := logger.Initialize("Info"); err != nil {
		panic(err)
	}
	logger.Log.Info("Starting agent")
	mc, err := storage.NewMetricsCollection()
	if err != nil {
		log.Fatal(err)
	}

	err = parseFlags()
	if err != nil {
		log.Fatal(err)
	}
	logger.Log.Info("parse flags success")

	ch := make(chan struct{})
	mc.UpdateValues(CliOpt.PollInterval, ch)

	for {
		time.Sleep(CliOpt.ReportInterval)
		err = retriableHttpSend(SendMetricsJSON, mc)
		if err != nil {
			close(ch)
			time.Sleep(2 * time.Second)
			log.Fatal(err)
		}
		err = retriableHttpSend(SendBatchJSON, mc)
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
	url := "http://" + CliOpt.NetAddr.String() + "/update/"
	for name, value := range cMetrics {
		mt := models.Metrics{
			ID:    name,
			MType: "counter",
			Delta: (*int64)(&value),
			Value: nil,
		}

		body, err := json.Marshal(mt)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		cBody, err := gzipCompress(body)
		if err != nil {
			return fmt.Errorf("failed to compress request body: %w", err)
		}

		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetHeader("Accept-Encoding", "gzip").
			SetBody(cBody).
			Post(url)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrCouldNotSendRequest, err)
		}
		if resp.StatusCode() != 200 {
			return ErrWrongResponseStatus
		}
	}
	for name, value := range gMetrics {
		mt := models.Metrics{
			ID:    name,
			MType: "gauge",
			Delta: nil,
			Value: (*float64)(&value),
		}
		body, err := json.Marshal(mt)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		cBody, err := gzipCompress(body)
		if err != nil {
			return fmt.Errorf("failed to compress request body: %w", err)
		}

		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetHeader("Accept-Encoding", "gzip").
			SetBody(cBody).
			Post(url)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrCouldNotSendRequest, err)
		}
		if resp.StatusCode() != 200 {
			logger.Log.Debug("response body",
				zap.String("resp body", string(resp.Body())))
			return ErrWrongResponseStatus
		}
	}
	return nil
}

func SendBatchJSON(mc storage.Collection) error {
	client := resty.New()
	gMetrics := mc.GetGaugeList()
	cMetrics := mc.GetCounterList()
	var allMetrics []models.Metrics

	url := "http://" + CliOpt.NetAddr.String() + "/updates/"

	for name, value := range cMetrics {
		mt := models.Metrics{
			ID:    name,
			MType: "counter",
			Delta: (*int64)(&value),
			Value: nil,
		}
		allMetrics = append(allMetrics, mt)
	}
	for name, value := range gMetrics {
		mt := models.Metrics{
			ID:    name,
			MType: "gauge",
			Delta: nil,
			Value: (*float64)(&value),
		}
		allMetrics = append(allMetrics, mt)
	}

	body, err := json.Marshal(allMetrics)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	cBody, err := gzipCompress(body)
	if err != nil {
		return fmt.Errorf("failed to compress request body: %w", err)
	}

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Accept-Encoding", "gzip").
		SetBody(cBody).
		Post(url)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCouldNotSendRequest, err)
	}
	if resp.StatusCode() != 200 {
		logger.Log.Info("", zap.Any("Body", string(resp.Body())))
		return ErrWrongResponseStatus
	}
	return nil
}

func gzipCompress(data []byte) ([]byte, error) {
	var buffer bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buffer, gzip.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("failed init compress writer: %v", err)
	}
	_, err = writer.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed write data to compress temporary buffer: %v", err)
	}
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed compress data: %v", err)
	}

	logger.Log.Info("Compression stats",
		zap.Int("Given", len(data)),
		zap.Int("Compressed", len(buffer.Bytes())))
	return buffer.Bytes(), nil
}

func gzipDecompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed init compress reader: %v", err)
	}
	defer reader.Close()

	var buffer bytes.Buffer
	_, err = buffer.ReadFrom(reader)
	if err != nil {
		return nil, fmt.Errorf("failed decompress data: %v", err)
	}
	return buffer.Bytes(), nil
}
