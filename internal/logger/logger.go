package logger

import (
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

var Log *zap.Logger = zap.NewNop()

func Initialize(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return fmt.Errorf("log level parsing: %v", err)
	}

	cfg := zap.NewDevelopmentConfig()
	cfg.Level = lvl
	zl, err := cfg.Build()
	if err != nil {
		return fmt.Errorf("zap initialization: %v", err)
	}
	Log = zl
	return nil
}

func HanlderWithLogger(h http.HandlerFunc) http.HandlerFunc {
	logFn := func(rw http.ResponseWriter, r *http.Request) {
		reqData := NewRequestData()
		reqData.Set(r.URL.Path, r.Method)
		Log.Info("Got request",
			zap.String("URI", reqData.url),
			zap.String("Method", reqData.method),
			zap.String("Content-Type", r.Header.Get("Content-Type")),
			zap.String("Accept-Encoding", r.Header.Get("Accept-Encoding")),
		)

		respData := NewResponseData()
		lw := NewLoggingResponseWriter(rw, respData)

		h.ServeHTTP(lw, r)
		Log.Info("Sending response",
			zap.Int("Status", respData.statusCode),
			zap.Int("Response Size", respData.respSizeB),
			zap.String("Content-Type", respData.respContentType),
			zap.String("Content-Encoding", respData.respContentEncoding),
		)
		Log.Info("Time spent processing request",
			zap.Any("Time spent", time.Since(reqData.timeStart)),
		)
	}
	return logFn
}
