package middleware

import (
	"time"

	"github.com/Fuonder/metriccoll.git/internal/logger"
	"github.com/Fuonder/metriccoll.git/internal/storage"
	"go.uber.org/zap"
)

type workerSendFunc func([]byte, string) error

func RetryableWorkerHTTPSend(sender workerSendFunc, remoteURL string, data []byte, retriesCount int) error {
	var err error
	timeouts := make([]time.Duration, retriesCount)
	for i := 0; i < retriesCount; i++ {
		timeouts[i] = time.Duration(2*i+1) * time.Second
	}
	//timeouts := []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

	for i := 0; i < retriesCount; i++ {
		logger.Log.Info("sending metrics")
		err = sender(data, remoteURL)
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

// senderFunc Deprecated
type senderFunc func(storage.Collection) error

// retriableHTTPSend Deprecated
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
