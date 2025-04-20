// Package server описывает функционал, который необходим для работы HTTP севера.
// В том числе различные HTTP endpoint'ы и дополнительные HTTP middleware функции.
package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Fuonder/metriccoll.git/internal/logger"
	"github.com/Fuonder/metriccoll.git/internal/models"
	"github.com/Fuonder/metriccoll.git/internal/storage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// ErrorResponse описывает структуру ошибки, которая возвращается в случае проблем при обработке запроса.
type ErrorResponse struct {
	Code    int    `json:"code"`              // HTTP-код ошибки.
	Message string `json:"message"`           // Человекочитаемое сообщение об ошибке.
	Details string `json:"details,omitempty"` // Зарезервированное поле с описанием ошибки. На данный момент не используется.
}

var (
	ErrMetricReaderNotInitialized      = ErrorResponse{Code: http.StatusInternalServerError, Message: "metricReader object not initialized"}
	ErrMetricWriterNotInitialized      = ErrorResponse{Code: http.StatusInternalServerError, Message: "metricWriter object not initialized"}
	ErrMetricFileHandlerNotInitialized = ErrorResponse{Code: http.StatusInternalServerError, Message: "metricFileHandler object not initialized"}
	ErrMetricDBHandlerNotInitialized   = ErrorResponse{Code: http.StatusInternalServerError, Message: "metricDatabaseHandler object not initialized"}
	ErrInvalidMetricValue              = errors.New("invalid metric value")
	ErrNoHashKey                       = errors.New("no hash key")
	ErrMismatchedHash                  = errors.New("mismatched hash")
)

// Handler реализует обработчики HTTP-запросов для различных endpoint-ов сервиса метрик.
type Handler struct {
	mReader      storage.MetricReader          // Интерфейс для чтения метрик.
	mWriter      storage.MetricWriter          // Интерфейс для записи метрик.
	mFileHandler storage.MetricFileHandler     // Интерфейс для работы с файлами.
	mDBHandler   storage.MetricDatabaseHandler // Интерфейс для взаимодействия с БД.
	hashKey      string                        // Ключ для проверки/генерации HMAC.
}

// NewHandler создает новый экземпляр Handler и инициализирует зависимости.
func NewHandler(mReader storage.MetricReader,
	mWriter storage.MetricWriter,
	mFileHandler storage.MetricFileHandler,
	mDBHandler storage.MetricDatabaseHandler,
	hashKey string) *Handler {
	h := Handler{
		mReader:      mReader,
		mWriter:      mWriter,
		mFileHandler: mFileHandler,
		mDBHandler:   mDBHandler,
		hashKey:      hashKey,
	}
	return &h
}

// RootHandler обрабатывает корневой GET-запрос и возвращает список всех метрик в text/HTML формате.
func (h *Handler) RootHandler(rw http.ResponseWriter, r *http.Request) {

	if h.mReader == nil {
		// TODO: consider using html template for error page
		rw.Header().Set("Content-Type", "text/html")
		rw.WriteHeader(ErrMetricReaderNotInitialized.Code)
		resp, _ := json.MarshalIndent(ErrMetricReaderNotInitialized, "", "    ")
		rw.Write(resp)
		return
	}

	logger.Log.Debug("Entering root handler")

	var metricList []models.Metrics
	logger.Log.Debug("creating metric list")

	metricList = h.mReader.GetAllMetrics()
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

	//if r.Header.Get("Accept-Encoding") == "gzip" {
	//	rw.Header().Set("Content-Encoding", "gzip")
	//}
	out := strings.Join(stringMetricList, ", ")
	rw.Header().Set("Content-Type", "text/html")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(out))
}

// ValueHandler возвращает значение конкретной метрики по имени и типу (gauge или counter).
func (h *Handler) ValueHandler(rw http.ResponseWriter, r *http.Request) {
	if h.mReader == nil {
		rw.WriteHeader(ErrMetricReaderNotInitialized.Code)
		resp, _ := json.MarshalIndent(ErrMetricReaderNotInitialized, "", "    ")
		rw.Write(resp)
		return
	}

	logger.Log.Debug("entering value handler")
	mType := chi.URLParam(r, "mType")
	mName := chi.URLParam(r, "mName")

	metric, err := h.mReader.GetMetricByName(mName, mType)
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

// UpdateHandler обновляет значение метрики, переданной в URL-параметрах.
func (h *Handler) UpdateHandler(rw http.ResponseWriter, r *http.Request) {
	if h.mWriter == nil {
		resp, _ := json.MarshalIndent(ErrMetricWriterNotInitialized, "", "    ")
		rw.WriteHeader(ErrMetricWriterNotInitialized.Code)
		rw.Write(resp)
		return
	}

	logger.Log.Debug("Updating metric")

	mType := chi.URLParam(r, "mType")
	mName := chi.URLParam(r, "mName")
	mValue := chi.URLParam(r, "mValue")

	if mType == "gauge" {
		value, _ := models.CheckTypeGauge(mValue)
		err := h.mWriter.AppendMetric(models.Metrics{ID: mName, MType: "gauge", Value: (*float64)(&value)})
		if err != nil {
			logger.Log.Info("can not add metric", zap.Error(err))
			if errors.Is(err, ErrInvalidMetricValue) {
				http.Error(rw, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(rw, "Internal server error", http.StatusInternalServerError)
			return
		}
		rw.WriteHeader(http.StatusOK)
	} else if mType == "counter" {
		value, _ := models.CheckTypeCounter(mValue)

		err := h.mWriter.AppendMetric(models.Metrics{ID: mName, MType: "counter", Delta: (*int64)(&value)})
		if err != nil {
			logger.Log.Info("can not add metric", zap.Error(err))
			if errors.Is(err, ErrInvalidMetricValue) {
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

// JSONUpdateHandler обновляет метрику, переданную в теле запроса в формате JSON.
// Возвращает обновленное значение метрики в ответе.
func (h *Handler) JSONUpdateHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	if h.mReader == nil {
		resp, _ := json.MarshalIndent(ErrMetricReaderNotInitialized, "", "    ")
		rw.WriteHeader(ErrMetricReaderNotInitialized.Code)
		rw.Write(resp)
		return
	}

	if h.mWriter == nil {
		resp, _ := json.MarshalIndent(ErrMetricWriterNotInitialized, "", "    ")
		rw.WriteHeader(ErrMetricWriterNotInitialized.Code)
		rw.Write(resp)
		return
	}

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
	err := h.mWriter.AppendMetric(mt)
	if err != nil {
		logger.Log.Debug("can not add metric", zap.Error(err))
		if errors.Is(err, ErrInvalidMetricValue) {
			logger.Log.Debug("invalid metric value")
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		logger.Log.Debug("other error then adding metric")
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	mtRes, err := h.mReader.GetMetricByName(mt.ID, mt.MType)
	if err != nil {
		logger.Log.Info("can not get metric by name", zap.Error(err))
		if errors.Is(err, ErrInvalidMetricValue) {
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

// JSONGetHandler возвращает значение метрики, переданной в теле запроса в формате JSON.
func (h *Handler) JSONGetHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	if h.mReader == nil {
		resp, _ := json.MarshalIndent(ErrMetricReaderNotInitialized, "", "    ")
		rw.WriteHeader(ErrMetricReaderNotInitialized.Code)
		rw.Write(resp)
		return
	}

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
	mt, err := h.mReader.GetMetricByName(metric.ID, metric.MType)
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

// DBPingHandler проверяет доступность соединения с базой данных.
// Используется для проверки состояния хранилища.
func (h *Handler) DBPingHandler(rw http.ResponseWriter, r *http.Request) {
	if h.mDBHandler == nil {
		resp, _ := json.MarshalIndent(ErrMetricDBHandlerNotInitialized, "", "    ")
		rw.WriteHeader(ErrMetricDBHandlerNotInitialized.Code)
		rw.Write(resp)
		return
	}

	err := h.mDBHandler.CheckConnection()
	if err != nil {
		logger.Log.Info("can not connect to database", zap.Error(err))
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(""))
}

// MultipleUpdateHandler обрабатывает пакетное обновление метрик, переданных в JSON-массиве.
// Возвращает обновленные значения всех переданных метрик.
func (h *Handler) MultipleUpdateHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	defer r.Body.Close()

	if h.mWriter == nil {
		rw.WriteHeader(ErrMetricWriterNotInitialized.Code)
		_ = json.NewEncoder(rw).Encode(ErrMetricWriterNotInitialized)
		return
	}
	if h.mReader == nil {
		rw.WriteHeader(ErrMetricReaderNotInitialized.Code)
		_ = json.NewEncoder(rw).Encode(ErrMetricReaderNotInitialized)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		logger.Log.Info("Invalid content type", zap.String("Content-Type", contentType))
		http.Error(rw, `{"error": "Invalid content type"}`, http.StatusBadRequest)
		return
	}

	var metrics []models.Metrics

	logger.Log.Info("DECODING BATCH")

	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		logger.Log.Debug("json decode error", zap.Error(err))
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	logger.Log.Info("APPENDING METRICS BATCH")

	if err := h.mWriter.AppendMetrics(metrics); err != nil {
		logger.Log.Info("can not add metrics", zap.Error(err))
		http.Error(rw, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	updatedMetrics := make([]models.Metrics, 0, len(metrics))

	logger.Log.Info("FORMING RESP METRICS BATCH")
	for _, mt := range metrics {
		mtRes, err := h.mReader.GetMetricByName(mt.ID, mt.MType)
		if err != nil {
			logger.Log.Info("can not get metric by name", zap.Error(err))
			http.Error(rw, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		updatedMetrics = append(updatedMetrics, mtRes)
	}
	logger.Log.Info("MARSHALING FINAL METRICS BATCH")
	resp, err := json.Marshal(updatedMetrics)
	if err != nil {
		logger.Log.Info("json marshal error", zap.Error(err))
		http.Error(rw, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write(resp)
}
