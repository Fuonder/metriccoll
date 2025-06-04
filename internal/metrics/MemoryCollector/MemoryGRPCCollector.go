package memcollector

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/certmanager"
	"github.com/Fuonder/metriccoll.git/internal/logger"
	"github.com/Fuonder/metriccoll.git/internal/metrics/middleware"
	"github.com/Fuonder/metriccoll.git/internal/storage"
	"github.com/Fuonder/metriccoll.git/proto"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/metadata"
)

type MemoryGRPCCollector struct {
	parent *MemoryCollector
	// st storage.Collection  					-> moved to parent

	//remoteIP      string
	//hashKey       string 						-> moved to parent
	//cipherManager certmanager.TLSCipher 		-> moved to parent
	//jobsCh        chan []byte  				-> moved to parent
	//tData         TimeIntervals  				-> moved to parent
	//wg            sync.WaitGroup  			-> moved to parent
	//localIP       string  					-> moved to parent
	client proto.MetricsClient
}

func (c *MemoryGRPCCollector) WaitWorkers() {
	c.parent.WaitWorkers()
}

func (c *MemoryGRPCCollector) SetStorage(stArg storage.Collection) error {
	return c.parent.SetStorage(stArg)
}

func (c *MemoryGRPCCollector) SetHashKey(key string) error {
	return c.parent.SetHashKey(key)
}

func (c *MemoryGRPCCollector) Collect(ctx context.Context, cancel context.CancelFunc) error {
	return c.parent.Collect(ctx, cancel)
}

func NewMemoryGRPCCollector(
	stArg storage.Collection,
	tData *TimeIntervals,
	jobsCh chan []byte,
	cipherManager certmanager.TLSCipher,
	client proto.MetricsClient) (*MemoryGRPCCollector, error) {
	logger.Log.Debug("Creating Memory Collector")

	// -> moved to parent
	//ip, err := getLocalIP()
	//if err != nil {
	//	return nil, fmt.Errorf("cannot get local ip: %v", err)
	//}

	parentCollector, err := NewMemoryCollector(stArg, tData, jobsCh, cipherManager)
	if err != nil {
		return nil, err
	}
	c := &MemoryGRPCCollector{parentCollector, client}

	// -> moved to parent
	//c := &MemoryGRPCCollector{st: stArg,
	//	//remoteIP:      "",
	//	hashKey:       "",
	//	cipherManager: cipherManager,
	//	jobsCh:        jobsCh,
	//	tData:         *tData,
	//	localIP:       ip,
	//	client:        client}

	return c, nil
}

func (c *MemoryGRPCCollector) Send(batch []byte) error {
	compressed, err := middleware.GzipCompress(batch)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	encrypted, err := c.parent.cipherManager.Cipher(compressed)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}

	logger.Log.Debug("creating metadata")
	md := metadata.New(map[string]string{
		"X-Real-IP":        c.parent.localIP,
		"content-encoding": "gzip",
	})
	logger.Log.Debug("creating metadata - OK")
	logger.Log.Debug("checking hash")
	if c.parent.hashKey != "" {
		h := hmac.New(sha256.New, []byte(c.parent.hashKey))
		h.Write(encrypted)
		hmacSum := base64.URLEncoding.EncodeToString(h.Sum(nil))
		md.Set("HashSHA256", hmacSum)
		logger.Log.Debug("hash written")
	}
	logger.Log.Debug("creating context ")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	logger.Log.Debug("Calling grpc", zap.Any("DATA", encrypted))
	resp, err := c.client.UpdateMetrics(ctx, &proto.EncryptedMessage{Blob: encrypted})
	if err != nil {
		logger.Log.Info("Got error during calling grpc", zap.Error(err))
		return err
	}
	if resp.Error != "" {
		logger.Log.Info("ERROR FROM GRPC SERVER", zap.Any("", resp.Error))
	}
	unzipped, err := middleware.GzipDecompress(resp.Blob)
	if err != nil {
		logger.Log.Info("Got error during decompressing Response", zap.Error(err))
		return err
	}
	logger.Log.Debug("RESPONSE", zap.Any("data", string(unzipped)))
	return err
}

func (c *MemoryGRPCCollector) RunWorkers(rateLimit int64) error {
	g := new(errgroup.Group)

	for i := 0; i < int(rateLimit); i++ {
		c.parent.wg.Add(1)
		workerID := i
		g.Go(func() error {
			defer c.parent.wg.Done()
			return c.worker(workerID, c.parent.jobsCh)

		})
	}

	if err := g.Wait(); err != nil {
		logger.Log.Debug("workers exited with error", zap.Error(err))
		return fmt.Errorf("method RunWorkers: %v", err)
	}
	return nil
}

func (c *MemoryGRPCCollector) worker(idx int, jobs <-chan []byte) error {
	for job := range jobs {
		logger.Log.Info("processing job", zap.Int("worker", idx))
		err := middleware.RetryableWorkerGRPCSend(c.Send, job, 3)
		if err != nil {
			logger.Log.Debug("sending batch failed", zap.Error(err))
			return fmt.Errorf("worker %d: %v", idx, err)
		}
	}
	return nil
}
