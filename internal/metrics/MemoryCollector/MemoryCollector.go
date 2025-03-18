package memcollector

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/logger"
	"github.com/Fuonder/metriccoll.git/internal/metrics/middleware"
	"github.com/Fuonder/metriccoll.git/internal/models"
	"github.com/Fuonder/metriccoll.git/internal/storage"
	"github.com/go-resty/resty/v2"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"net/http"
	"sync"
	"time"
)

var (
	ErrCouldNotSendRequest = errors.New("could not send request")
	ErrWrongResponseStatus = errors.New("wrong request data or metrics value")
)

type TimeIntervals struct {
	reportInterval time.Duration
	pollInterval   time.Duration
}

func NewTimeIntervals(rInterval time.Duration, pInterval time.Duration) *TimeIntervals {
	return &TimeIntervals{
		reportInterval: rInterval,
		pollInterval:   pInterval,
	}
}

type MemoryCollector struct {
	st       storage.Collection
	remoteIP string
	hashKey  string
	jobsCh   chan []byte
	tData    TimeIntervals
}

func NewMemoryCollector(stArg storage.Collection, tData *TimeIntervals, jobsCh chan []byte) *MemoryCollector {
	logger.Log.Debug("Creating Memory Collector")
	c := &MemoryCollector{st: stArg,
		remoteIP: "",
		hashKey:  "",
		jobsCh:   jobsCh,
		tData:    *tData}
	return c
}

//func NewEmptyMemoryCollector() *MemoryCollector {
//	logger.Log.Debug("Creating Empty Memory Collector")
//	c := &MemoryCollector{st: nil, remoteIP: "", hashKey: "", jobsCh: make(chan []byte, 10)}
//	return c
//}

func (c *MemoryCollector) SetStorage(stArg storage.Collection) error {
	if c.st != nil {
		logger.Log.Warn("Changing existing storage")
	}
	c.st = stArg
	return nil
}

func (c *MemoryCollector) SetRemoteIP(remoteIP string) error {
	if c.remoteIP != "" {
		logger.Log.Warn("Changing existing remoteIP")
	}
	c.remoteIP = remoteIP
	return nil
}

func (c *MemoryCollector) SetHashKey(key string) error {
	c.hashKey = key
	return nil
}

func getMemoryInfo() ([]models.Metrics, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return []models.Metrics{}, err
	}

	var mList []models.Metrics

	totalMemoryFloat := float64(v.Total)
	freeMemoryFloat := float64(v.Available)
	mList = append(mList, models.Metrics{
		ID:    "TotalMemory",
		MType: "gauge",
		Delta: nil,
		Value: &totalMemoryFloat,
	})
	mList = append(mList, models.Metrics{
		ID:    "FreeMemory",
		MType: "gauge",
		Delta: nil,
		Value: &freeMemoryFloat,
	})

	return mList, nil

}

func getCPUUtilization() ([]models.Metrics, error) {
	var mList []models.Metrics
	percentages, _ := cpu.Percent(0, true)

	for idx, p := range percentages {
		mt := models.Metrics{
			ID:    fmt.Sprintf("CPUutilization%d", idx),
			MType: "gauge",
			Delta: nil,
			Value: &p,
		}
		mList = append(mList, mt)
	}
	return mList, nil
}

func (c *MemoryCollector) collectNewMetrics(ctx context.Context) ([]byte, error) {

	select {
	case <-ctx.Done():
		return []byte{}, ctx.Err()
	default:
		time.Sleep(c.tData.reportInterval)
		var allMetrics []models.Metrics

		mCPUUtilization, err := getCPUUtilization()
		if err != nil {
			return []byte{}, err
		}

		allMetrics = append(allMetrics, mCPUUtilization...)

		mMemory, err := getMemoryInfo()
		if err != nil {
			return []byte{}, err
		}

		allMetrics = append(allMetrics, mMemory...)

		body, err := json.Marshal(allMetrics)

		if err != nil {
			return []byte{}, err
		}
		return body, nil
	}

}

func (c *MemoryCollector) collectOriginalMetrics(ctx context.Context) ([]byte, error) {
	select {
	case <-ctx.Done():
		logger.Log.Info("Stopping collection")
		return []byte{}, ctx.Err()
	default:
		time.Sleep(c.tData.reportInterval)

		gMetrics := c.st.GetGaugeList()
		cMetrics := c.st.GetCounterList()
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

		body, err := json.Marshal(allMetrics)
		if err != nil {
			return []byte{}, fmt.Errorf("failed to marshal request body: %w", err)
		}
		return body, nil
	}

}

func (c *MemoryCollector) Collect(ctx context.Context, cancel context.CancelFunc) error {
	g := new(errgroup.Group)

	c.st.UpdateValues(ctx, c.tData.pollInterval)

	g.Go(func() error {
		for {
			data, err := c.collectOriginalMetrics(ctx)
			if err != nil {
				cancel()
				return fmt.Errorf("collect orig: %v", err)
			}
			c.jobsCh <- data
		}
	})

	g.Go(func() error {
		for {
			data, err := c.collectNewMetrics(ctx)
			if err != nil {
				cancel()
				return fmt.Errorf("collect new: %v", err)
			}
			c.jobsCh <- data
		}
	})

	if err := g.Wait(); err != nil {
		logger.Log.Info("collect exit with error")
		return err
	}
	return nil

}

func (c *MemoryCollector) RunWorkers(rateLimit int64) error {
	var wg sync.WaitGroup
	g := new(errgroup.Group)

	for i := range int(rateLimit) {
		wg.Add(1)
		g.Go(func() error {
			err := c.worker(i, c.jobsCh, &wg)
			if err != nil {
				return err
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		logger.Log.Debug("workers exited with error", zap.Error(err))
		return fmt.Errorf("method RunWorkers: %v", err)
	}
	return nil
}

func (c *MemoryCollector) Post(packetBody []byte, remoteURL string) error {
	if remoteURL == "" {
		remoteURL = "http://" + c.remoteIP + "/updates/"
	}

	client := resty.New()

	cBody, err := middleware.GzipCompress(packetBody)
	if err != nil {
		return fmt.Errorf("failed to compress request body: %w", err)
	}

	var resp *resty.Response

	if c.hashKey != "" {
		logger.Log.Info("Creating HMAC")
		h := hmac.New(sha256.New, []byte(c.hashKey))
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
			Post(remoteURL)
	} else {
		logger.Log.Info("Sending batch")
		resp, err = client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetHeader("Accept-Encoding", "gzip").
			SetBody(cBody).
			Post(remoteURL)
	}

	if err != nil {
		return fmt.Errorf("%w: %v", ErrCouldNotSendRequest, err)
	}
	if resp.StatusCode() != 200 {
		logger.Log.Info("", zap.Any("Body", string(resp.Body())))
		return ErrWrongResponseStatus
	}
	logger.Log.Info("Request sent successfully")
	return nil
}

func (c *MemoryCollector) CheckConnection() error {
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get("http://" + c.remoteIP)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned non-OK status: %d", resp.StatusCode)
	}

	return nil
}

func (c *MemoryCollector) worker(idx int, jobs <-chan []byte, wg *sync.WaitGroup) error {
	defer wg.Done()
	for job := range jobs {
		logger.Log.Info("processing job", zap.Int("worker", idx))
		err := middleware.RetryableWorkerHTTPSend(c.Post, "", job, 3)
		if err != nil {
			logger.Log.Debug("sending batch failed", zap.Error(err))
			return fmt.Errorf("worker %d: %v", idx, err)
		}
	}
	return nil
}
