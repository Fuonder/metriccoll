package server

import (
	"bytes"
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

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"` // for future possible use
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

type Handler struct {
	mReader      storage.MetricReader
	mWriter      storage.MetricWriter
	mFileHandler storage.MetricFileHandler
	mDBHandler   storage.MetricDatabaseHandler
	hashKey      string
}

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

//func GzipMiddleware(h http.HandlerFunc) http.HandlerFunc {
//	return func(rw http.ResponseWriter, r *http.Request) {

func (h *Handler) WithHashing(next http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		hw := rw
		if h.hashKey != "" {
			hw = newHashWriter(rw, h.hashKey)
		}
		next.ServeHTTP(hw, r)
	}
}

//func (h *Handler) DecompressRequestMiddleware(next http.Handler) http.Handler {
//	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		if strings.EqualFold(r.Header.Get("Content-Encoding"), "gzip") {
//			// берём gzip.Reader из пула
//			gr := gzipReaderPool.Get().(*gzip.Reader)
//			if err := gr.Reset(r.Body); err != nil {
//				gzipReaderPool.Put(gr) // вернуть обратно в пул, даже если ошибка
//				http.Error(w, "Failed to reset gzip reader", http.StatusBadRequest)
//				return
//			}
//
//			// заменяем тело запроса на распакованный поток
//			r.Body = &pooledGzipBody{
//				Reader: gr,
//				Closer: r.Body,
//				pool:   &gzipReaderPool,
//			}
//		}
//
//		next.ServeHTTP(w, r)
//	})
//}
