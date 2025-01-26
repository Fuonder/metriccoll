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
	logger.Log.Info("Entering root handler")

	rw.Header().Set("Content-Type", "text/html")
	var metricList []models.Metrics
	logger.Log.Info("creating metric list")

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
	logger.Log.Info("final metric list",
		zap.String("metrics", strings.Join(stringMetricList, ", ")))
	io.WriteString(rw, strings.Join(stringMetricList, ", "))
}

func (h *Handler) ValueHandler(rw http.ResponseWriter, r *http.Request) {
	logger.Log.Info("entering value handler")
	mType := chi.URLParam(r, "mType")
	mName := chi.URLParam(r, "mName")

	metric, err := h.storage.GetMetricByName(mName, mType)
	if err != nil {
		logger.Log.Error("get metric by name error", zap.Error(err))
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
	logger.Log.Info("Updating metric")

	mType := chi.URLParam(r, "mType")
	mName := chi.URLParam(r, "mName")
	mValue := chi.URLParam(r, "mValue")

	if mType == "gauge" {
		value, _ := models.CheckTypeGauge(mValue)
		err := h.storage.AppendMetric(models.Metrics{ID: mName, MType: "gauge", Value: (*float64)(&value)})
		if err != nil {
			logger.Log.Error("can not add metric", zap.Error(err))
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
			logger.Log.Error("can not add metric", zap.Error(err))
			if errors.Is(err, storage.ErrInvalidMetricValue) {
				http.Error(rw, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(rw, "Internal server error", http.StatusInternalServerError)
			return
		}
		rw.WriteHeader(http.StatusOK)
	} else {
		logger.Log.Error("Invalid metric type, can not add metric")
		http.Error(rw, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) JSONUpdateHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	logger.Log.Info("entering json update handler")
	if r.Header.Get("Content-Type") != "application/json" {
		logger.Log.Info("invalid content type",
			zap.String("Content-Type", r.Header.Get("Content-Type")))
		http.Error(rw, "Invalid content type", http.StatusBadRequest)
		return
	}
	var mt models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&mt); err != nil {
		logger.Log.Info("json decode error", zap.Error(err))
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	fmt.Println(mt)

	err := h.storage.AppendMetric(mt)
	if err != nil {
		logger.Log.Info("can not add metric", zap.Error(err))
		if errors.Is(err, storage.ErrInvalidMetricValue) {
			logger.Log.Info("invalid metric value")
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		logger.Log.Info("over error then adding metric")
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	mtRes, err := h.storage.GetMetricByName(mt.ID, mt.MType)
	if err != nil {
		logger.Log.Error("can not get metric by name", zap.Error(err))
		if errors.Is(err, storage.ErrInvalidMetricValue) {
			logger.Log.Info("invalid metric value")
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		logger.Log.Info("over error then getting metric")
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	resp, err := json.MarshalIndent(mtRes, "", "    ")
	if err != nil {
		logger.Log.Info("json marshal error", zap.Error(err))
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(resp)
}

func (h *Handler) JSONGetHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	if r.Header.Get("Content-Type") != "application/json" {
		logger.Log.Info("Invalid content type !!!!",
			zap.String("Content-Type", r.Header.Get("Content-Type")))
		http.Error(rw, "Invalid content type", http.StatusBadRequest)
	}
	var metric models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		logger.Log.Info("Can not parse json request", zap.Error(err))
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Println(metric)
	defer r.Body.Close()
	mt, err := h.storage.GetMetricByName(metric.ID, metric.MType)
	if err != nil {
		logger.Log.Info("metric not found", zap.Error(err))
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}
	metric = mt
	resp, err := json.MarshalIndent(metric, "", "    ")
	if err != nil {
		logger.Log.Info("can not create response", zap.Error(err))
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	rw.WriteHeader(http.StatusOK)
	logger.Log.Info("SENDING RESPONSE", zap.String("resp", string(resp)))
	rw.Write(resp)
}

func (h *Handler) CheckMethod(next http.Handler) http.Handler {
	logger.Log.Info("checking method")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			logger.Log.Error("wrong method", zap.String("method", r.Method))
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		} else {
			logger.Log.Info("method - OK")
			next.ServeHTTP(w, r)
		}
	})
}
func (h *Handler) CheckContentType(next http.Handler) http.Handler {
	logger.Log.Info("checking content type")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" &&
			r.Header.Get("Content-Type") != "text/plain" &&
			r.Header.Get("Content-Type") != "text/plain; charset=UTF-8" &&
			r.Header.Get("Content-Type") != "text/plain; charset=utf-8" &&
			r.Header.Get("Content-Type") != "" {
			logger.Log.Error("wrong content type",
				zap.String("Content-Type", r.Header.Get("Content-Type")))
			http.Error(w, "invalid content type", http.StatusBadRequest)
		} else {
			logger.Log.Info("content type - OK")
			next.ServeHTTP(w, r)
		}

	})
}
func (h *Handler) CheckMetricType(next http.Handler) http.Handler {
	logger.Log.Info("checking metric type")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mType := chi.URLParam(r, "mType")
		if mType != "counter" && mType != "gauge" {
			logger.Log.Error("wrong metric type",
				zap.String("Type", mType))
			http.Error(w, "invalid metric type", http.StatusBadRequest)
		} else {
			logger.Log.Info("metric type - OK")
			next.ServeHTTP(w, r)
		}
	})
}

func (h *Handler) CheckMetricName(next http.Handler) http.Handler {
	logger.Log.Info("checking metric name")
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		mName := chi.URLParam(r, "mName")
		if strings.TrimSpace(mName) == "" {
			logger.Log.Error("empty metric name")
			http.Error(rw, "metric name is required", http.StatusNotFound)
		} else {
			logger.Log.Info("metric name - OK")
			next.ServeHTTP(rw, r)
		}
	})
}

func (h *Handler) CheckMetricValue(next http.Handler) http.Handler {
	logger.Log.Info("checking metric value")
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		mType := chi.URLParam(r, "mType")
		mValue := chi.URLParam(r, "mValue")
		var err error
		logger.Log.Info("guessing metric type")
		if mType == "gauge" {
			_, err = models.CheckTypeGauge(mValue)
		} else if mType == "counter" {
			_, err = models.CheckTypeCounter(mValue)
		}
		if err != nil {
			logger.Log.Error("invalid metric value",
				zap.Any("value", mValue))
			http.Error(rw, "invalid metric value", http.StatusBadRequest)
		} else {
			logger.Log.Info("metric value - OK")
			next.ServeHTTP(rw, r)
		}
	})
}
