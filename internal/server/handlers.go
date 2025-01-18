package server

import (
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/logger"
	model "github.com/Fuonder/metriccoll.git/internal/models"
	"github.com/Fuonder/metriccoll.git/internal/storage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type Handler struct {
	storage storage.Storage
}

func NewHandler(storage storage.Storage) *Handler {
	return &Handler{storage: storage}
}

func (h *Handler) RootHandler(rw http.ResponseWriter, r *http.Request) {
	logger.Log.Debug("Entering root handler")

	rw.Header().Set("Content-Type", "text/html")
	var metricList []string
	logger.Log.Debug("creating metric list")

	gMetrics := h.storage.GetGaugeList()
	cMetrics := h.storage.GetCounterList()
	for name, value := range gMetrics {
		metricList = append(metricList, fmt.Sprintf("%s %s",
			name,
			strconv.FormatFloat(float64(value), 'f', -1, 64)))
	}
	for name, value := range cMetrics {
		metricList = append(metricList, fmt.Sprintf("%s %s",
			name,
			strconv.FormatInt(int64(value), 10)))
	}
	logger.Log.Debug("final metric list",
		zap.String("metrics", strings.Join(metricList, ", ")))
	io.WriteString(rw, strings.Join(metricList, ", "))
}

func (h *Handler) ValueHandler(rw http.ResponseWriter, r *http.Request) {
	logger.Log.Debug("entering value handler")
	mType := chi.URLParam(r, "mType")
	mName := chi.URLParam(r, "mName")

	if mType == "gauge" {
		value, err := h.storage.GetGaugeMetric(mName)
		if err != nil {
			logger.Log.Error("gauge metric error", zap.Error(err))
			http.Error(rw, err.Error(), http.StatusNotFound)
		} else {
			io.WriteString(rw, strconv.FormatFloat(float64(value), 'f', -1, 64))
			return
		}
	} else if mType == "counter" {
		value, err := h.storage.GetCounterMetric(mName)
		if err != nil {
			logger.Log.Error("counter metric error", zap.Error(err))
			http.Error(rw, err.Error(), http.StatusNotFound)
		} else {
			logger.Log.Debug("leaving value handler")
			io.WriteString(rw, strconv.FormatInt(int64(value), 10))
			return
		}
	}
}
func (h *Handler) UpdateHandler(rw http.ResponseWriter, r *http.Request) {
	logger.Log.Debug("Updating metric")

	mType := chi.URLParam(r, "mType")
	mName := chi.URLParam(r, "mName")
	mValue := chi.URLParam(r, "mValue")

	if mType == "gauge" {
		value, _ := model.CheckTypeGauge(mValue)
		h.storage.AppendGaugeMetric(mName, value)
		rw.WriteHeader(http.StatusOK)
	} else if mType == "counter" {
		value, _ := model.CheckTypeCounter(mValue)
		h.storage.AppendCounterMetric(mName, value)
		rw.WriteHeader(http.StatusOK)
	} else {
		logger.Log.Error("Invalid metric type, can not add metric")
		http.Error(rw, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) CheckMethod(next http.Handler) http.Handler {
	logger.Log.Debug("checking method")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			logger.Log.Error("wrong method", zap.String("method", r.Method))
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		} else {
			logger.Log.Debug("method - OK")
			next.ServeHTTP(w, r)
		}
	})
}
func (h *Handler) CheckContentType(next http.Handler) http.Handler {
	logger.Log.Debug("checking content type")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "text/plain" &&
			r.Header.Get("Content-Type") != "text/plain; charset=UTF-8" &&
			r.Header.Get("Content-Type") != "text/plain; charset=utf-8" &&
			r.Header.Get("Content-Type") != "" {
			logger.Log.Error("wrong content type",
				zap.String("Content-Type", r.Header.Get("Content-Type")))
			http.Error(w, "invalid content type", http.StatusBadRequest)
		} else {
			logger.Log.Debug("content type - OK")
			next.ServeHTTP(w, r)
		}
	})
}
func (h *Handler) CheckMetricType(next http.Handler) http.Handler {
	logger.Log.Debug("checking metric type")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mType := chi.URLParam(r, "mType")
		if mType != "counter" && mType != "gauge" {
			logger.Log.Error("wrong metric type",
				zap.String("Type", mType))
			http.Error(w, "invalid metric type", http.StatusBadRequest)
		} else {
			logger.Log.Debug("metric type - OK")
			next.ServeHTTP(w, r)
		}
	})
}

func (h *Handler) CheckMetricName(next http.Handler) http.Handler {
	logger.Log.Debug("checking metric name")
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		mName := chi.URLParam(r, "mName")
		if strings.TrimSpace(mName) == "" {
			logger.Log.Error("empty metric name")
			http.Error(rw, "metric name is required", http.StatusNotFound)
		} else {
			logger.Log.Debug("metric name - OK")
			next.ServeHTTP(rw, r)
		}
	})
}

func (h *Handler) CheckMetricValue(next http.Handler) http.Handler {
	logger.Log.Debug("checking metric value")
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		mType := chi.URLParam(r, "mType")
		mValue := chi.URLParam(r, "mValue")
		var err error
		logger.Log.Debug("guessing metric type")
		if mType == "gauge" {
			_, err = model.CheckTypeGauge(mValue)
		} else if mType == "counter" {
			_, err = model.CheckTypeCounter(mValue)
		}
		if err != nil {
			logger.Log.Error("invalid metric value",
				zap.Any("value", mValue))
			http.Error(rw, "invalid metric value", http.StatusBadRequest)
		} else {
			logger.Log.Debug("metric value - OK")
			next.ServeHTTP(rw, r)
		}
	})
}
