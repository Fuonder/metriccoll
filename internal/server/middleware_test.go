package server

import (
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckMethod(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		expected int
	}{
		{"ValidGET", http.MethodGet, http.StatusOK},
		{"ValidPOST", http.MethodPost, http.StatusOK},
		{"InvalidMethod", http.MethodPut, http.StatusMethodNotAllowed},
		{"InvalidDELETE", http.MethodDelete, http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Use((&Handler{}).CheckMethod)
			r.Get("/", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
			r.Post("/", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

			req := httptest.NewRequest(tt.method, "/", nil)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			require.Equal(t, tt.expected, rr.Code)
		})
	}
}

func TestCheckContentType(t *testing.T) {
	tests := []struct {
		name         string
		contentType  string
		expectedCode int
	}{
		{"ValidContentType", "text/plain", http.StatusOK},
		{"ValidContentTypeCharsetUTF8", "text/plain; charset=UTF-8", http.StatusOK},
		{"ValidContentTypeCharsetUtf8", "text/plain; charset=utf-8", http.StatusOK},
		{"ValidJSONContentType", "application/json", http.StatusOK},
		{"InvalidContentType", "application/xml", http.StatusBadRequest},
		{"EmptyContentType", "", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Use((&Handler{}).CheckContentType)
			r.Post("/", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.Header.Set("Content-Type", tt.contentType)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			require.Equal(t, tt.expectedCode, rr.Code)
		})
	}
}

func TestCheckMetricType(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		expectedCode int
	}{
		{"ValidGauge", "/update/gauge/metric/1.23", http.StatusOK},
		{"ValidCounter", "/update/counter/metric/10", http.StatusOK},
		{"InvalidMetricType", "/update/unknown/metric/10", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()

			r.Route("/update", func(r chi.Router) {
				r.Route("/{mType}", func(r chi.Router) {
					r.Use((&Handler{}).CheckMetricType)
					r.Post("/{mName}/{mValue}", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
				})
			})

			req := httptest.NewRequest(http.MethodPost, tt.url, nil)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			require.Equal(t, tt.expectedCode, rr.Code)
		})
	}
}

func TestCheckMetricName(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		expectedCode int
	}{
		{"ValidMetricName", "/update/gauge/test/1.23", http.StatusOK},
		{"EmptyMetricName", "/update/gauge//1.23", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Route("/update", func(r chi.Router) {
				r.Route("/{mType}", func(r chi.Router) {
					r.Route("/{mName}", func(r chi.Router) {
						r.Use((&Handler{}).CheckMetricName)
						r.Post("/{mValue}", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
					})
				})
			})

			req := httptest.NewRequest(http.MethodPost, tt.url, nil)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			require.Equal(t, tt.expectedCode, rr.Code)
		})
	}
}

func TestCheckMetricValue(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		expectedCode int
	}{
		{"ValidGaugeValue", "/update/gauge/metric/1.23", http.StatusOK},
		{"InvalidGaugeValue", "/update/gauge/metric/invalid", http.StatusBadRequest},
		{"ValidCounterValue", "/update/counter/metric/10", http.StatusOK},
		{"InvalidCounterValue", "/update/counter/metric/invalid", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Route("/update", func(r chi.Router) {
				r.Route("/{mType}", func(r chi.Router) {
					r.Route("/{mName}", func(r chi.Router) {
						r.Route("/{mValue}", func(r chi.Router) {
							r.Use((&Handler{}).CheckMetricValue)
							r.Post("/", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
						})
					})
				})
			})

			req := httptest.NewRequest(http.MethodPost, tt.url, nil)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			require.Equal(t, tt.expectedCode, rr.Code)
		})
	}
}
