package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/logger"
	"github.com/Fuonder/metriccoll.git/internal/models"
	"github.com/Fuonder/metriccoll.git/internal/storage"
	"github.com/go-resty/resty/v2"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var (
	ErrCouldNotSendRequest = errors.New("could not send request")
	ErrWrongResponseStatus = errors.New("wrong request data or metrics value")
)

type senderFunc func(storage.Collection) error
type workerSendFunc func([]byte) error

func retriableHTTPSend(sender senderFunc, st storage.Collection) error {
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

func retriableWorkerHTTPSend(sender workerSendFunc, data []byte) error {
	var err error
	timeouts := []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		logger.Log.Info("sending metrics")
		err = sender(data)
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

	ctx, cancel := context.WithCancel(context.Background())
	mc.UpdateValues(ctx, CliOpt.PollInterval) // goroutine which collects original metrics and saves them in storage.Collection
	defer cancel()

	g := new(errgroup.Group)

	/*
			1. for workers in range RATE_LIMIT -> goroutine: create workers (workers listen jobs channel)
			2. infinite loop:
		    	2.1 time.Sleep Report Interval
				2.2 collect original metrics and write to jobs channel
				2.3 collect new metrics and write to jobs channel
	*/

	jobsCh := make(chan []byte, 10)
	defer close(jobsCh)

	var wg sync.WaitGroup
	for i := range CliOpt.RateLimit {
		wg.Add(1)
		g.Go(func() error {
			err = worker(i, jobsCh, &wg)
			if err != nil {
				return err
			}
			return nil
		})
	}

	go collectOriginalMetrics(jobsCh, mc)
	go collectNewMetrics(jobsCh)
	/*
		TODO: need to implement stop channel or context and use this method
		g.Go(func() error {
			err = collectOriginalMetrics(jobsCh, mc)
			if err != nil {
				return err
			}
			return nil
		})
			g.Go(func() error {
				err = collectNewMetrics(jobsCh)
				if err != nil {
					return err
				}
				return nil
			})
	*/

	if err := g.Wait(); err != nil {
		logger.Log.Info("agent exited with error", zap.Error(err))
		cancel()
		panic(err)
	}

	//err = testAll()
	//if err != nil {
	//	close(ch)
	//	time.Sleep(2 * time.Second)
	//	log.Fatal(err)
	//}
	//}
}

func worker(idx int64, jobs <-chan []byte, wg *sync.WaitGroup) error {
	defer wg.Done()
	for job := range jobs {
		logger.Log.Info("processing job", zap.Int64("worker", idx))
		err := retriableWorkerHTTPSend(SendBatch, job)
		if err != nil {
			logger.Log.Info("sending batch failed", zap.Error(err))
			return err
		}
	}
	return nil
}

// TODO: errors + canceling
func collectOriginalMetrics(jobs chan []byte, mc storage.Collection) {
	//var err error
	for {
		time.Sleep(CliOpt.ReportInterval)
		gMetrics := mc.GetGaugeList()
		cMetrics := mc.GetCounterList()
		allMetrics := []models.Metrics{}

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

		body, _ := json.Marshal(allMetrics)
		//if err != nil {
		//	return fmt.Errorf("failed to marshal request body: %w", err)
		//}
		jobs <- body
	}

}

// TODO: errors + canceling
func collectNewMetrics(jobs chan []byte) {
	for {
		time.Sleep(CliOpt.ReportInterval)
		v, _ := mem.VirtualMemory()
		percentages, _ := cpu.Percent(0, true)
		totalMemoryFloat := float64(v.Total)
		freeMemoryFloat := float64(v.Available)
		allMetrics := []models.Metrics{}

		for idx, p := range percentages {
			mt := models.Metrics{
				ID:    fmt.Sprintf("CPUutilization%d", idx),
				MType: "gauge",
				Delta: nil,
				Value: &p,
			}
			allMetrics = append(allMetrics, mt)
		}

		allMetrics = append(allMetrics, models.Metrics{
			ID:    "TotalMemory",
			MType: "gauge",
			Delta: nil,
			Value: &totalMemoryFloat,
		})
		allMetrics = append(allMetrics, models.Metrics{
			ID:    "FreeMemory",
			MType: "gauge",
			Delta: nil,
			Value: &freeMemoryFloat,
		})
		fmt.Println(allMetrics)
		body, _ := json.Marshal(allMetrics)
		jobs <- body
	}
}

func SendBatch(data []byte) error {
	client := resty.New()
	url := "http://" + CliOpt.NetAddr.String() + "/updates/"

	cBody, err := gzipCompress(data)
	if err != nil {
		return fmt.Errorf("failed to compress request body: %w", err)
	}

	var resp *resty.Response

	if CliOpt.HashKey != "" {
		logger.Log.Info("Creating HMAC")
		h := hmac.New(sha256.New, []byte(CliOpt.HashKey))
		h.Write(cBody)
		s := h.Sum(nil)
		logger.Log.Info("HASH", zap.String("HASH", base64.URLEncoding.EncodeToString(s)))
		logger.Log.Info("Writing HMAC")
		logger.Log.Info("Sending batch with HMAC")
		resp, err = client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetHeader("Accept-Encoding", "gzip").
			SetHeader("HashSHA256", base64.URLEncoding.EncodeToString(s)).
			SetBody(cBody).
			Post(url)
	} else {
		logger.Log.Info("Sending batch")
		resp, err = client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetHeader("Accept-Encoding", "gzip").
			SetBody(cBody).
			Post(url)
	}

	if err != nil {
		return fmt.Errorf("%w: %v", ErrCouldNotSendRequest, err)
	}
	if resp.StatusCode() != 200 {
		logger.Log.Info("", zap.Any("Body", string(resp.Body())))
		return ErrWrongResponseStatus
	}
	return nil
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

	var resp *resty.Response

	if CliOpt.HashKey != "" {
		logger.Log.Info("Creating HMAC")
		h := hmac.New(sha256.New, []byte(CliOpt.HashKey))
		h.Write(cBody)
		s := h.Sum(nil)
		logger.Log.Info("HASH", zap.String("HASH", base64.URLEncoding.EncodeToString(s)))
		logger.Log.Info("Writing HMAC")
		logger.Log.Info("Sending batch with HMAC")
		resp, err = client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetHeader("Accept-Encoding", "gzip").
			SetHeader("HashSHA256", base64.URLEncoding.EncodeToString(s)).
			SetBody(cBody).
			Post(url)
	} else {
		logger.Log.Info("Sending batch")
		resp, err = client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetHeader("Accept-Encoding", "gzip").
			SetBody(cBody).
			Post(url)
	}

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
