package logger

import (
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"time"
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
		Log.Debug("Got request",
			zap.String("URI", reqData.url),
			zap.String("Method", reqData.method),
		)

		respData := NewResponseData()
		lw := NewLoggingResponseWriter(rw, respData)

		h.ServeHTTP(lw, r)
		Log.Debug("Sending response",
			zap.Int("Status", respData.statusCode),
			zap.Int("Response Size", respData.respSizeB),
		)
		Log.Debug("Time spent processing request",
			zap.Any("Time spent", time.Since(reqData.timeStart)),
		)
	}
	return logFn
}
