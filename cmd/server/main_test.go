package main

import (
	"github.com/Fuonder/metriccoll.git/internal/server"
	"github.com/Fuonder/metriccoll.git/internal/storage"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testRequest(t *testing.T, ts *httptest.Server,
	method string, contentType string, path string) (*http.Response, string) {

	req, err := http.NewRequest(method, ts.URL+path, nil)
	require.NoError(t, err)
	req.Header.Set("Content-Type", contentType)
	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func TestMetricRouter(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		method      string
		contentType string
		want        int
		wantResp    string
	}{
		{
			name:        "PositiveGauge",
			url:         "/update/gauge/gMetric/1.01",
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
			contentType: "application/json111",
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

	testsGet := []struct {
		name        string
		url         string
		method      string
		contentType string
		want        int
		wantResp    string
	}{
		{
			name:        "PositiveGetValue",
			url:         "/value/gauge/gMetric",
			method:      http.MethodGet,
			contentType: "text/plain",
			want:        http.StatusOK,
			wantResp:    "1.01",
		},
		{
			name:        "PositiveGetAllValues",
			url:         "/",
			method:      http.MethodGet,
			contentType: "text/plain",
			want:        http.StatusOK,
			wantResp:    "gMetric 1.01, cMetric 2",
		},
		{name: "NegativeValue",
			url:         "/value/gauge/negative",
			method:      http.MethodGet,
			contentType: "text/plain",
			want:        http.StatusNotFound,
			wantResp:    "metric with such key is not found: negative\n",
		},
	}
	ms, err := storage.NewJSONStorage()
	h := server.NewHandler(ms)
	require.NoError(t, err)
	ts := httptest.NewServer(metricRouter(h))
	defer ts.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp, _ := testRequest(t, ts, test.method, test.contentType, test.url)
			defer resp.Body.Close()
			require.Equal(t, test.want, resp.StatusCode)
		})
	}
	for _, test := range testsGet {
		t.Run(test.name, func(t *testing.T) {
			resp, body := testRequest(t, ts, test.method, test.contentType, test.url)
			defer resp.Body.Close()
			require.Equal(t, test.want, resp.StatusCode)
			require.Equal(t, test.wantResp, body)
		})
	}
}
