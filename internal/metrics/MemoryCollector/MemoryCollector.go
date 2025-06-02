package memcollector

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/certmanager"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Fuonder/metriccoll.git/internal/logger"
	"github.com/Fuonder/metriccoll.git/internal/metrics/middleware"
	"github.com/Fuonder/metriccoll.git/internal/models"
	"github.com/Fuonder/metriccoll.git/internal/storage"
	"github.com/go-resty/resty/v2"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
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
	st            storage.Collection
	remoteIP      string
	hashKey       string
	cipherManager certmanager.TLSCipher
	jobsCh        chan []byte
	tData         TimeIntervals
	wg            sync.WaitGroup
	localIP       string
}

func NewMemoryCollector(stArg storage.Collection,
	tData *TimeIntervals,
	jobsCh chan []byte,
	cipherManager certmanager.TLSCipher) (*MemoryCollector, error) {
	logger.Log.Debug("Creating Memory Collector")
	ip, err := getLocalIP()
	if err != nil {
		return nil, fmt.Errorf("cannot get local ip: %v", err)
	}
	c := &MemoryCollector{st: stArg,
		remoteIP:      "",
		hashKey:       "",
		cipherManager: cipherManager,
		jobsCh:        jobsCh,
		tData:         *tData,
		localIP:       ip}
	return c, nil
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
		return nil, ctx.Err()
	default:
		time.Sleep(c.tData.reportInterval)
		cpuMetrics, err := getCPUUtilization()
		if err != nil {
			return nil, err
		}
		memMetrics, err := getMemoryInfo()
		if err != nil {
			return nil, err
		}
		all := append(cpuMetrics, memMetrics...)
		return json.Marshal(all)
	}
}

func (c *MemoryCollector) collectOriginalMetrics(ctx context.Context) ([]byte, error) {
	select {
	case <-ctx.Done():
		logger.Log.Info("Stopping collection")
		return nil, ctx.Err()
	default:
		time.Sleep(c.tData.reportInterval)
		var all []models.Metrics
		for k, v := range c.st.GetCounterList() {
			val := int64(v)
			all = append(all, models.Metrics{
				ID:    k,
				MType: "counter",
				Delta: &val,
			})
		}
		for k, v := range c.st.GetGaugeList() {
			val := float64(v)
			all = append(all, models.Metrics{
				ID:    k,
				MType: "gauge",
				Value: &val,
			})
		}
		return json.Marshal(all)
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
				if errors.Is(err, context.Canceled) {
					return nil
				}
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
				if errors.Is(err, context.Canceled) {
					return nil
				}
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
	g := new(errgroup.Group)

	for i := 0; i < int(rateLimit); i++ {
		c.wg.Add(1)
		workerID := i
		g.Go(func() error {
			defer c.wg.Done()
			return c.worker(workerID, c.jobsCh)

		})
	}

	if err := g.Wait(); err != nil {
		logger.Log.Debug("workers exited with error", zap.Error(err))
		return fmt.Errorf("method RunWorkers: %v", err)
	}
	return nil
}

func (c *MemoryCollector) WaitWorkers() {
	c.wg.Wait()
}

func (c *MemoryCollector) Post(packetBody []byte, remoteURL string) error {
	if remoteURL == "" {
		remoteURL = "http://" + c.remoteIP + "/updates/"
	}
	client := resty.New()
	cBody, err := middleware.GzipCompress(packetBody)
	if err != nil {
		return fmt.Errorf("compress failed: %w", err)
	}
	cBody, err = c.cipherManager.Cipher(cBody)
	if err != nil {
		return fmt.Errorf("cipher failed: %w", err)
	}

	req := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Accept-Encoding", "gzip").
		SetHeader("X-Real-IP", c.localIP).
		SetBody(cBody)

	if c.hashKey != "" {
		h := hmac.New(sha256.New, []byte(c.hashKey))
		h.Write(cBody)
		hash := base64.URLEncoding.EncodeToString(h.Sum(nil))
		req.SetHeader("HashSHA256", hash)
	}

	resp, err := req.Post(remoteURL)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCouldNotSendRequest, err)
	}
	if resp.StatusCode() != http.StatusOK {
		return ErrWrongResponseStatus
	}
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

func (c *MemoryCollector) worker(idx int, jobs <-chan []byte) error {
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

func getLocalIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, i := range interfaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			if ip.To4() != nil {
				return ip.String(), nil
			}
		}
	}
	return "", fmt.Errorf("no connected network interface found")
}
