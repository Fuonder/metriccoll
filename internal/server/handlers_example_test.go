package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/models"
	storage "github.com/Fuonder/metriccoll.git/internal/storage/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"net/http"
	"net/http/httptest"
)

func ExampleHandler_RootHandler() {
	ctrl := gomock.NewController(nil)
	defer ctrl.Finish()

	mockReader := storage.NewMockMetricReader(ctrl)
	mockReader.EXPECT().GetAllMetrics().Return([]models.Metrics{
		{ID: "PollCount", MType: "counter", Delta: models.Int64Ptr(11639078)},
		{ID: "BuckHashSys", MType: "gauge", Value: models.Float64Ptr(3301)},
		{ID: "GCSys", MType: "gauge", Value: models.Float64Ptr(2491368)},
		{ID: "HeapReleased", MType: "gauge", Value: models.Float64Ptr(2400256)},
		{ID: "HeapSys", MType: "gauge", Value: models.Float64Ptr(7847936)},
		{ID: "Sys", MType: "gauge", Value: models.Float64Ptr(11754512)},
		{ID: "MCacheInuse", MType: "gauge", Value: models.Float64Ptr(3600)},
		{ID: "PauseTotalNs", MType: "gauge", Value: models.Float64Ptr(1417573)},
		{ID: "HeapAlloc", MType: "gauge", Value: models.Float64Ptr(1515480)},
		{ID: "HeapObjects", MType: "gauge", Value: models.Float64Ptr(1518)},
		{ID: "OtherSys", MType: "gauge", Value: models.Float64Ptr(823059)},
		{ID: "StackInuse", MType: "gauge", Value: models.Float64Ptr(524288)},
		{ID: "RandomValue", MType: "gauge", Value: models.Float64Ptr(29.719607371392165)},
		{ID: "Alloc", MType: "gauge", Value: models.Float64Ptr(1515480)},
		{ID: "Frees", MType: "gauge", Value: models.Float64Ptr(2303)},
		{ID: "LastGC", MType: "gauge", Value: models.Float64Ptr(1745266266262774500)}})

	h := NewHandler(mockReader, nil, nil, nil, nil, "")

	r := chi.NewRouter()
	r.Get("/", h.RootHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	fmt.Println(rec.Code)
	fmt.Println(rec.Body.String())

	// Output:
	// 200
	// PollCount 11639078, BuckHashSys 3301, GCSys 2491368, HeapReleased 2400256, HeapSys 7847936, Sys 11754512, MCacheInuse 3600, PauseTotalNs 1417573, HeapAlloc 1515480, HeapObjects 1518, OtherSys 823059, StackInuse 524288, RandomValue 29.719607371392165, Alloc 1515480, Frees 2303, LastGC 1745266266262774500
}

func ExampleHandler_DBPingHandler() {
	ctrl := gomock.NewController(nil)
	defer ctrl.Finish()

	mockDBHandler := storage.NewMockMetricDatabaseHandler(ctrl)
	mockDBHandler.EXPECT().CheckConnection().Return(nil)

	h := NewHandler(nil, nil, nil, mockDBHandler, nil, "")

	r := chi.NewRouter()
	r.Get("/ping", h.DBPingHandler)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	fmt.Println(rec.Code)
	fmt.Println(rec.Body.String())

	// Output:
	// 200
}

func ExampleHandler_MultipleUpdateHandler() {
	ctrl := gomock.NewController(nil)
	defer ctrl.Finish()

	mockWriter := storage.NewMockMetricWriter(ctrl)
	mockReader := storage.NewMockMetricReader(ctrl)

	metrics := []models.Metrics{
		{ID: "PollCount", MType: "counter", Delta: models.Int64Ptr(11639078)},
		{ID: "BuckHashSys", MType: "gauge", Value: models.Float64Ptr(3301)},
		{ID: "GCSys", MType: "gauge", Value: models.Float64Ptr(2491368)},
		{ID: "HeapReleased", MType: "gauge", Value: models.Float64Ptr(2400256)},
		{ID: "HeapSys", MType: "gauge", Value: models.Float64Ptr(7847936)},
		{ID: "Sys", MType: "gauge", Value: models.Float64Ptr(11754512)},
		{ID: "MCacheInuse", MType: "gauge", Value: models.Float64Ptr(3600)},
		{ID: "PauseTotalNs", MType: "gauge", Value: models.Float64Ptr(1417573)},
		{ID: "HeapAlloc", MType: "gauge", Value: models.Float64Ptr(1515480)},
		{ID: "HeapObjects", MType: "gauge", Value: models.Float64Ptr(1518)},
		{ID: "OtherSys", MType: "gauge", Value: models.Float64Ptr(823059)},
		{ID: "StackInuse", MType: "gauge", Value: models.Float64Ptr(524288)},
		{ID: "RandomValue", MType: "gauge", Value: models.Float64Ptr(29.719607371392165)},
		{ID: "Alloc", MType: "gauge", Value: models.Float64Ptr(1515480)},
		{ID: "Frees", MType: "gauge", Value: models.Float64Ptr(2303)},
		{ID: "LastGC", MType: "gauge", Value: models.Float64Ptr(1745266266262774500)},
	}

	mockWriter.EXPECT().AppendMetrics(metrics).Return(nil)

	for _, mt := range metrics {
		mockReader.EXPECT().GetMetricByName(mt.ID, mt.MType).Return(mt, nil)
	}

	h := NewHandler(mockReader, mockWriter, nil, nil, nil, "")

	r := chi.NewRouter()
	r.Post("/updates/", h.MultipleUpdateHandler)

	body, _ := json.MarshalIndent(metrics, "", "    ")
	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, rec.Body.Bytes(), "", "    "); err != nil {
		fmt.Println("Failed to format JSON:", err)
		return
	}
	fmt.Println(rec.Code)
	fmt.Println(prettyJSON.String())

	// Output:
	// 200
	// [
	//     {
	//         "id": "PollCount",
	//         "type": "counter",
	//         "delta": 11639078
	//     },
	//     {
	//         "id": "BuckHashSys",
	//         "type": "gauge",
	//         "value": 3301
	//     },
	//     {
	//         "id": "GCSys",
	//         "type": "gauge",
	//         "value": 2491368
	//     },
	//     {
	//         "id": "HeapReleased",
	//         "type": "gauge",
	//         "value": 2400256
	//     },
	//     {
	//         "id": "HeapSys",
	//         "type": "gauge",
	//         "value": 7847936
	//     },
	//     {
	//         "id": "Sys",
	//         "type": "gauge",
	//         "value": 11754512
	//     },
	//     {
	//         "id": "MCacheInuse",
	//         "type": "gauge",
	//         "value": 3600
	//     },
	//     {
	//         "id": "PauseTotalNs",
	//         "type": "gauge",
	//         "value": 1417573
	//     },
	//     {
	//         "id": "HeapAlloc",
	//         "type": "gauge",
	//         "value": 1515480
	//     },
	//     {
	//         "id": "HeapObjects",
	//         "type": "gauge",
	//         "value": 1518
	//     },
	//     {
	//         "id": "OtherSys",
	//         "type": "gauge",
	//         "value": 823059
	//     },
	//     {
	//         "id": "StackInuse",
	//         "type": "gauge",
	//         "value": 524288
	//     },
	//     {
	//         "id": "RandomValue",
	//         "type": "gauge",
	//         "value": 29.719607371392165
	//     },
	//     {
	//         "id": "Alloc",
	//         "type": "gauge",
	//         "value": 1515480
	//     },
	//     {
	//         "id": "Frees",
	//         "type": "gauge",
	//         "value": 2303
	//     },
	//     {
	//         "id": "LastGC",
	//         "type": "gauge",
	//         "value": 1745266266262774500
	//     }
	// ]
}

func ExampleHandler_ValueHandler() {
	ctrl := gomock.NewController(nil)
	defer ctrl.Finish()

	mockReader := storage.NewMockMetricReader(ctrl)

	gaugeValue := 29.719607371392165
	mockReader.EXPECT().
		GetMetricByName("RandomValue", "gauge").
		Return(models.Metrics{
			ID:    "RandomValue",
			MType: "gauge",
			Value: &gaugeValue,
		}, nil)

	handler := NewHandler(mockReader, nil, nil, nil, nil, "")

	r := chi.NewRouter()
	r.Get("/value/{mType}/{mName}", handler.ValueHandler)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/RandomValue", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	fmt.Println(rec.Code)
	fmt.Println(rec.Body.String())

	// Output:
	// 200
	// 29.719607371392165
}

func ExampleHandler_UpdateHandler() {
	ctrl := gomock.NewController(nil)
	defer ctrl.Finish()

	mockWriter := storage.NewMockMetricWriter(ctrl)

	mockWriter.EXPECT().
		AppendMetric(models.Metrics{
			ID:    "HeapAlloc",
			MType: "gauge",
			Value: models.Float64Ptr(123.456),
		}).
		Return(nil)

	handler := NewHandler(nil, mockWriter, nil, nil, nil, "")

	r := chi.NewRouter()
	r.Post("/update/{mType}/{mName}/{mValue}", handler.UpdateHandler)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/HeapAlloc/123.456", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	fmt.Println(rec.Code)

	// Output:
	// 200
}

func ExampleHandler_JSONUpdateHandler() {
	ctrl := gomock.NewController(nil)
	defer ctrl.Finish()

	mockWriter := storage.NewMockMetricWriter(ctrl)
	mockReader := storage.NewMockMetricReader(ctrl)

	metric := models.Metrics{
		ID:    "HeapAlloc",
		MType: "gauge",
		Value: func() *float64 { v := 456.78; return &v }(),
	}

	mockWriter.EXPECT().
		AppendMetric(metric).
		Return(nil)

	mockReader.EXPECT().
		GetMetricByName("HeapAlloc", "gauge").
		Return(metric, nil)

	handler := NewHandler(mockReader, mockWriter, nil, nil, nil, "")

	body, _ := json.Marshal(metric)
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.JSONUpdateHandler(rec, req)

	fmt.Println(rec.Code)
	fmt.Println(rec.Body.String())

	// Output:
	// 200
	// {"id":"HeapAlloc","type":"gauge","value":456.78}
}

func ExampleHandler_JSONGetHandler() {
	ctrl := gomock.NewController(nil)
	defer ctrl.Finish()

	mockReader := storage.NewMockMetricReader(ctrl)

	metric := models.Metrics{
		ID:    "HeapAlloc",
		MType: "gauge",
		Value: func() *float64 { v := 123.45; return &v }(),
	}

	mockReader.EXPECT().
		GetMetricByName("HeapAlloc", "gauge").
		Return(metric, nil)

	handler := NewHandler(mockReader, nil, nil, nil, nil, "")

	requestBody, _ := json.Marshal(models.Metrics{
		ID:    "HeapAlloc",
		MType: "gauge",
	})

	req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.JSONGetHandler(rec, req)

	fmt.Println(rec.Code)
	fmt.Println(rec.Body.String())

	// Output:
	// 200
	// {
	//     "id": "HeapAlloc",
	//     "type": "gauge",
	//     "value": 123.45
	// }
}
