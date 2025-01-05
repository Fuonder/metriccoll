package main

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetricHandler(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		method      string
		contentType string
		want        int
	}{
		{
			name:        "PositiveGauge",
			url:         "/update/gauge/gMetric/1.00",
			method:      http.MethodPost,
			contentType: "text/plain",
			want:        http.StatusOK,
		},
		{
			name:        "PositiveCounter",
			url:         "/update/counter/cMetric/2",
			method:      http.MethodPost,
			contentType: "text/plain",
			want:        http.StatusOK,
		},
		{
			name:        "NegativeWrongMethod",
			url:         "/update/gauge/gMetric/3.00",
			method:      http.MethodDelete,
			contentType: "text/plain",
			want:        http.StatusMethodNotAllowed,
		},
		{
			name:        "NegativeWrongContentType",
			url:         "/update/counter/cMetric/4",
			method:      http.MethodPost,
			contentType: "application/json",
			want:        http.StatusBadRequest,
		},
		{
			name:        "NegativeWrongUrl",
			url:         "/update111111/gauge/gMetric/3.00",
			method:      http.MethodPost,
			contentType: "text/plain",
			want:        http.StatusNotFound,
		},
		{
			name:        "NegativeNoMetricValue",
			url:         "/update/counter/cMetric/",
			method:      http.MethodPost,
			contentType: "text/plain",
			want:        http.StatusBadRequest,
		},
		{
			name:        "NegativeNoMetricName",
			url:         "/update/counter/",
			method:      http.MethodPost,
			contentType: "text/plain",
			want:        http.StatusNotFound,
		},
		{
			name:        "NegativeWrongMetricTypeName",
			url:         "/update/counter1/cMetric/10",
			method:      http.MethodPost,
			contentType: "text/plain",
			want:        http.StatusBadRequest,
		},
		{
			name:        "NegativeWrongGaugeValue",
			url:         "/update/gauge/gMetric/AA",
			method:      http.MethodPost,
			contentType: "text/plain",
			want:        http.StatusBadRequest,
		},
		{
			name:        "NegativeWrongCounterValue",
			url:         "/update/counter/cMetric/!!!",
			method:      http.MethodPost,
			contentType: "text/plain",
			want:        http.StatusBadRequest,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			//srvUrl := "http://localhost:8080"
			request := httptest.NewRequest(test.method, test.url, nil)
			request.Header.Set("Content-Type", test.contentType)
			recorder := httptest.NewRecorder()
			h := http.HandlerFunc(metricHandler)
			h(recorder, request)
			res := recorder.Result()
			defer res.Body.Close()

			require.Equal(t, test.want, res.StatusCode)

		})
	}

}
