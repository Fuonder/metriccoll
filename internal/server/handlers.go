package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/logger"
	"github.com/Fuonder/metriccoll.git/internal/models"
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
	var metricList []models.Metrics
	logger.Log.Debug("creating metric list")

	metricList = h.storage.GetAllMetrics()
	var stringMetricList []string

	for _, m := range metricList {
		if m.MType == "gauge" {
			stringMetricList = append(stringMetricList, fmt.Sprintf("%s %s",
				m.ID,
				strconv.FormatFloat(*m.Value, 'f', -1, 64)))
		} else if m.MType == "counter" {
			stringMetricList = append(stringMetricList, fmt.Sprintf("%s %s",
				m.ID,
				strconv.FormatInt(*m.Delta, 10)))
		}
	}
	logger.Log.Debug("final metric list",
		zap.String("metrics", strings.Join(stringMetricList, ", ")))
	io.WriteString(rw, strings.Join(stringMetricList, ", "))
}

func (h *Handler) ValueHandler(rw http.ResponseWriter, r *http.Request) {
	logger.Log.Debug("entering value handler")
	mType := chi.URLParam(r, "mType")
	mName := chi.URLParam(r, "mName")

	metric, err := h.storage.GetMetricByName(mName, mType)
	if err != nil {
		logger.Log.Info("get metric by name error", zap.Error(err))
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}
	if metric.MType == "gauge" {
		io.WriteString(rw, strconv.FormatFloat(*metric.Value, 'f', -1, 64))
		return
	} else if metric.MType == "counter" {
		io.WriteString(rw, strconv.FormatInt(*metric.Delta, 10))
		return
	}
}
func (h *Handler) UpdateHandler(rw http.ResponseWriter, r *http.Request) {
	logger.Log.Debug("Updating metric")

	mType := chi.URLParam(r, "mType")
	mName := chi.URLParam(r, "mName")
	mValue := chi.URLParam(r, "mValue")

	if mType == "gauge" {
		value, _ := models.CheckTypeGauge(mValue)
		err := h.storage.AppendMetric(models.Metrics{ID: mName, MType: "gauge", Value: (*float64)(&value)})
		if err != nil {
			logger.Log.Info("can not add metric", zap.Error(err))
			if errors.Is(err, storage.ErrInvalidMetricValue) {
				http.Error(rw, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(rw, "Internal server error", http.StatusInternalServerError)
			return
		}
		rw.WriteHeader(http.StatusOK)
	} else if mType == "counter" {
		value, _ := models.CheckTypeCounter(mValue)

		err := h.storage.AppendMetric(models.Metrics{ID: mName, MType: "counter", Delta: (*int64)(&value)})
		if err != nil {
			logger.Log.Info("can not add metric", zap.Error(err))
			if errors.Is(err, storage.ErrInvalidMetricValue) {
				http.Error(rw, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(rw, "Internal server error", http.StatusInternalServerError)
			return
		}
		rw.WriteHeader(http.StatusOK)
	} else {
		logger.Log.Info("Invalid metric type, can not add metric")
		http.Error(rw, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) JSONUpdateHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	logger.Log.Debug("entering json update handler")
	if r.Header.Get("Content-Type") != "application/json" {
		logger.Log.Info("invalid content type",
			zap.String("Content-Type", r.Header.Get("Content-Type")))
		http.Error(rw, "Invalid content type", http.StatusBadRequest)
		return
	}
	var mt models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&mt); err != nil {
		logger.Log.Debug("json decode error", zap.Error(err))
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	err := h.storage.AppendMetric(mt)
	if err != nil {
		logger.Log.Debug("can not add metric", zap.Error(err))
		if errors.Is(err, storage.ErrInvalidMetricValue) {
			logger.Log.Debug("invalid metric value")
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		logger.Log.Debug("other error then adding metric")
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	mtRes, err := h.storage.GetMetricByName(mt.ID, mt.MType)
	if err != nil {
		logger.Log.Info("can not get metric by name", zap.Error(err))
		if errors.Is(err, storage.ErrInvalidMetricValue) {
			logger.Log.Info("invalid metric value")
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		logger.Log.Info("other error then getting metric")
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := json.Marshal(mtRes)
	if err != nil {
		logger.Log.Info("json marshal error", zap.Error(err))
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	} else {
		logger.Log.Info("Marshaling ok - sending respinse with status 200")
		rw.WriteHeader(http.StatusOK)
		rw.Write(resp)
	}

}

func (h *Handler) JSONGetHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	if r.Header.Get("Content-Type") != "application/json" {
		logger.Log.Info("Invalid content type",
			zap.String("Content-Type", r.Header.Get("Content-Type")))
		http.Error(rw, "Invalid content type", http.StatusBadRequest)
	}
	var metric models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		logger.Log.Info("Can not parse json request", zap.Error(err))
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	mt, err := h.storage.GetMetricByName(metric.ID, metric.MType)
	if err != nil {
		logger.Log.Info("metric not found", zap.Error(err))
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}
	resp, err := json.MarshalIndent(mt, "", "    ")
	if err != nil {
		logger.Log.Info("can not create response", zap.Error(err))
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	rw.WriteHeader(http.StatusOK)
	rw.Write(resp)
}

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
func (h *Handler) CheckContentType(next http.Handler) http.Handler {
	logger.Log.Debug("checking content type")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" &&
			r.Header.Get("Content-Type") != "text/plain" &&
			r.Header.Get("Content-Type") != "text/plain; charset=UTF-8" &&
			r.Header.Get("Content-Type") != "text/plain; charset=utf-8" &&
			r.Header.Get("Content-Type") != "" {
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
		} else {
			logger.Log.Debug("metric value - OK")
			next.ServeHTTP(rw, r)
		}
	})
}

func GzipMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ow := rw
		if r.Header.Get("Content-Type") != "application/json" &&
			r.Header.Get("Content-Type") != "text/html" {
			h.ServeHTTP(ow, r)
			return
		}
		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
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
