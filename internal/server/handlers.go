package server

import (
	"fmt"
	model "github.com/Fuonder/metriccoll.git/internal/models"
	"github.com/Fuonder/metriccoll.git/internal/storage"
	"github.com/go-chi/chi/v5"
	"io"
	"log"
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
	log.Println("Entering root handler")

	rw.Header().Set("Content-Type", "text/html")
	var metricList []string
	log.Println("creating metric list")

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
	log.Printf("final metric list: %v", metricList)
	io.WriteString(rw, strings.Join(metricList, ", "))
}

func (h *Handler) ValueHandler(rw http.ResponseWriter, r *http.Request) {
	log.Println("entering value handler")
	mType := chi.URLParam(r, "mType")
	mName := chi.URLParam(r, "mName")

	if mType == "gauge" {
		value, err := h.storage.GetGaugeMetric(mName)
		if err != nil {
			log.Println(err)
			http.Error(rw, err.Error(), http.StatusNotFound)
		} else {
			io.WriteString(rw, strconv.FormatFloat(float64(value), 'f', -1, 64))
			return
		}
	} else if mType == "counter" {
		value, err := h.storage.GetCounterMetric(mName)
		if err != nil {
			log.Println(err)
			http.Error(rw, err.Error(), http.StatusNotFound)
		} else {
			log.Println("leaving value handler")
			io.WriteString(rw, strconv.FormatInt(int64(value), 10))
			return
		}
	}
}
func (h *Handler) UpdateHandler(rw http.ResponseWriter, r *http.Request) {
	log.Println("Updating metric")

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
		log.Println("Invalid metric type, can not add metric")
		http.Error(rw, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) CheckMethod(next http.Handler) http.Handler {
	log.Println("checking method")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			log.Printf("wrong method: %s", r.Method)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		} else {
			log.Println("method - OK")
			next.ServeHTTP(w, r)
		}
	})
}
func (h *Handler) CheckContentType(next http.Handler) http.Handler {
	log.Println("checking content type")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "text/plain" &&
			r.Header.Get("Content-Type") != "text/plain; charset=UTF-8" &&
			r.Header.Get("Content-Type") != "text/plain; charset=utf-8" &&
			r.Header.Get("Content-Type") != "" {
			log.Printf("wrong content type: %s", r.Header.Get("Content-Type"))
			http.Error(w, "invalid content type", http.StatusBadRequest)
		} else {
			log.Println("content type - OK")
			next.ServeHTTP(w, r)
		}
	})
}
func (h *Handler) CheckMetricType(next http.Handler) http.Handler {
	log.Println("checking metric type")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mType := chi.URLParam(r, "mType")
		if mType != "counter" && mType != "gauge" {
			log.Printf("wrong metric type: %s", mType)
			http.Error(w, "invalid metric type", http.StatusBadRequest)
		} else {
			log.Println("metric type - OK")
			next.ServeHTTP(w, r)
		}
	})
}

func (h *Handler) CheckMetricName(next http.Handler) http.Handler {
	log.Println("checking metric name")
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		mName := chi.URLParam(r, "mName")
		if strings.TrimSpace(mName) == "" {
			log.Printf("empty metric name: %s", mName)
			http.Error(rw, "metric name is required", http.StatusNotFound)
		} else {
			log.Println("metric name - OK")
			next.ServeHTTP(rw, r)
		}
	})
}

func (h *Handler) CheckMetricValue(next http.Handler) http.Handler {
	log.Println("checking metric value")
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		mType := chi.URLParam(r, "mType")
		mValue := chi.URLParam(r, "mValue")
		var err error
		log.Printf("guessing metric type")
		if mType == "gauge" {
			_, err = model.CheckTypeGauge(mValue)
		} else if mType == "counter" {
			_, err = model.CheckTypeCounter(mValue)
		}
		if err != nil {
			log.Printf("invalid metric value: %s", mValue)
			http.Error(rw, "invalid metric value", http.StatusBadRequest)
		} else {
			log.Println("metric value - OK")
			next.ServeHTTP(rw, r)
		}
	})
}
