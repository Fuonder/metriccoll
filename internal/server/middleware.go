package server

import (
	"bytes"
	"github.com/Fuonder/metriccoll.git/internal/logger"
	"github.com/Fuonder/metriccoll.git/internal/models"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strings"
)

var validContentTypes = map[string]struct{}{
	"text/plain":                {},
	"text/plain; charset=UTF-8": {},
	"text/plain; charset=utf-8": {},
	"application/json":          {},
}

func isValidContentType(ct string) bool {
	_, ok := validContentTypes[ct]
	return ok || ct == ""
}

// CheckMethod проверяет, что метод запроса является GET или POST.
func (h *Handler) CheckMethod(next http.Handler) http.Handler {
	logger.Log.Debug("checking method")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			logger.Log.Info("wrong method", zap.String("method", r.Method))
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		} else {
			logger.Log.Debug("method - OK")
			next.ServeHTTP(w, r)
		}
	})
}

// CheckContentType валидирует заголовок Content-Type входящего запроса.
func (h *Handler) CheckContentType(next http.Handler) http.Handler {
	logger.Log.Debug("checking content type")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isValidContentType(r.Header.Get("Content-Type")) {
			logger.Log.Info("wrong content type",
				zap.String("Content-Type", r.Header.Get("Content-Type")))
			http.Error(w, "invalid content type", http.StatusBadRequest)
			return
		} else {
			logger.Log.Debug("content type - OK")
			next.ServeHTTP(w, r)
		}

	})
}

// CheckMetricType проверяет тип метрики, переданный в URL-параметре.
// Допустимые значения: "gauge", "counter".
func (h *Handler) CheckMetricType(next http.Handler) http.Handler {
	logger.Log.Debug("checking metric type")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mType := chi.URLParam(r, "mType")
		if mType != "counter" && mType != "gauge" {
			logger.Log.Info("wrong metric type",
				zap.String("Type", mType))
			http.Error(w, "invalid metric type", http.StatusBadRequest)
			return
		} else {
			logger.Log.Debug("metric type - OK")
			next.ServeHTTP(w, r)
		}
	})
}

// CheckMetricName проверяет, что имя метрики присутствует в URL и не является пустым.
func (h *Handler) CheckMetricName(next http.Handler) http.Handler {
	logger.Log.Debug("checking metric name")
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		mName := chi.URLParam(r, "mName")
		if strings.TrimSpace(mName) == "" {
			logger.Log.Info("empty metric name")
			http.Error(rw, "metric name is required", http.StatusNotFound)
			return
		} else {
			logger.Log.Debug("metric name - OK")
			next.ServeHTTP(rw, r)
		}
	})
}

// CheckMetricValue проверяет корректность значения метрики в зависимости от типа.
func (h *Handler) CheckMetricValue(next http.Handler) http.Handler {
	logger.Log.Debug("checking metric value")
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		mType := chi.URLParam(r, "mType")
		mValue := chi.URLParam(r, "mValue")
		var err error
		logger.Log.Debug("guessing metric type")
		if mType == "gauge" {
			_, err = models.CheckTypeGauge(mValue)
		} else if mType == "counter" {
			_, err = models.CheckTypeCounter(mValue)
		}
		if err != nil {
			logger.Log.Info("invalid metric value",
				zap.Any("value", mValue))
			http.Error(rw, "invalid metric value", http.StatusBadRequest)
			return
		}
		logger.Log.Debug("metric value - OK")
		next.ServeHTTP(rw, r)
	})
}

// GzipMiddleware обеспечивает сжатие и/или распаковку тела запроса/ответа с использованием GZIP,
// если это поддерживается клиентом.
func GzipMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ow := rw
		acceptEncoding := r.Header.Get("Accept-Encoding")
		logger.Log.Info("GZIP: AcceptEncoding", zap.String("Accept-Encoding", acceptEncoding))
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		logger.Log.Info("GZIP: AcceptEncoding GZIP?", zap.Bool("SupportGZIP", supportsGzip))
		if supportsGzip {
			cw := newGzipWriter(rw)
			ow = cw
			ow.Header().Set("Content-Encoding", "gzip")
			defer cw.Close()
		}
		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			cr, err := newGzipReader(r.Body)
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer cr.Close()
		}
		h.ServeHTTP(ow, r)

	}
}

// HashMiddleware проверяет подпись HMAC (если задан ключ) и отклоняет запросы с некорректной подписью.
func (h *Handler) HashMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if h.hashKey == "" {
			next.ServeHTTP(rw, r)
			return
		}
		logger.Log.Info("Validating HMAC")
		if HMACPresent := r.Header.Get("HashSHA256"); HMACPresent != "" {
			var bodyCopy bytes.Buffer
			teeReader := io.TeeReader(r.Body, &bodyCopy)
			body, err := io.ReadAll(teeReader)
			if err != nil {
				http.Error(rw, "Error reading request body", http.StatusInternalServerError)
				return
			}
			err = validateHMAC(HMACPresent, body, h.hashKey)
			if err != nil {
				http.Error(rw, ErrMismatchedHash.Error(), http.StatusBadRequest)
				return
			}
			logger.Log.Info("Validation", zap.String("HMAC", "CORRECT"))
			r.Body = io.NopCloser(&bodyCopy)
		} else {
			logger.Log.Info("Validation", zap.String("HMAC", "No HMAC in request found, skipping validation"))
		}
		next.ServeHTTP(rw, r)
	})
}

// WithHashing добавляет подпись HMAC к ответу сервера, если задан ключ.
func (h *Handler) WithHashing(next http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		hw := rw
		if h.hashKey != "" {
			hw = newHashWriter(rw, h.hashKey)
		}
		next.ServeHTTP(hw, r)
	}
}
