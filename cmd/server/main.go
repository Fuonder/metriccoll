package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var ms, _ = NewMemStorage()

func main() {
	log.Println("Starting metric collector")
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func valueHandler(rw http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "mType")
	mName := chi.URLParam(r, "mName")

	if mType == "gauge" {
		value, err := ms.GetGaugeMetric(mName)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusNotFound)
		} else {
			io.WriteString(rw, mName+" "+strconv.FormatFloat(float64(value), 'f', -1, 64))
			return
		}
	} else if mType == "counter" {
		value, err := ms.GetCounterMetric(mName)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusNotFound)
		} else {
			io.WriteString(rw, mName+" "+strconv.FormatInt(int64(value), 10))
			return
		}
	}
}
func updateHandler(rw http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "mType")
	mName := chi.URLParam(r, "mName")
	mValue := chi.URLParam(r, "mValue")

	if mType == "gauge" {
		value, _ := checkTypeGauge(mValue)
		ms.AppendGaugeMetric(mName, value)
		rw.WriteHeader(http.StatusOK)
	} else if mType == "counter" {
		value, _ := checkTypeCounter(mValue)
		ms.AppendCounterMetric(mName, value)
		rw.WriteHeader(http.StatusOK)
	} else {
		http.Error(rw, "Internal server error", http.StatusInternalServerError)
	}
}
func rootHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "text/html")
	var metricList []string
	for name, value := range ms.gMetric {
		metricList = append(metricList, fmt.Sprintf("%s %s",
			name,
			strconv.FormatFloat(float64(value), 'f', -1, 64)))
	}
	for name, value := range ms.cMetric {
		metricList = append(metricList, fmt.Sprintf("%s %s",
			name,
			strconv.FormatInt(int64(value), 10)))
	}
	io.WriteString(rw, strings.Join(metricList, ", "))
}

func checkMethod(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}
func checkContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "text/plain" &&
			r.Header.Get("Content-Type") != "text/plain; charset=UTF-8" &&
			r.Header.Get("Content-Type") != "text/plain; charset=utf-8" {
			http.Error(w, "invalid content type", http.StatusBadRequest)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}
func checkMetricType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mType := chi.URLParam(r, "mType")
		if mType != "counter" && mType != "gauge" {
			http.Error(w, "invalid metric type", http.StatusBadRequest)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

func checkMetricName(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		mName := chi.URLParam(r, "mName")
		if strings.TrimSpace(mName) == "" {
			http.Error(rw, "metric name is required", http.StatusNotFound)
		} else {
			next.ServeHTTP(rw, r)
		}
	})
}

func checkMetricValue(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		mType := chi.URLParam(r, "mType")
		mValue := chi.URLParam(r, "mValue")
		var err error
		if mType == "gauge" {
			_, err = checkTypeGauge(mValue)
		} else if mType == "counter" {
			_, err = checkTypeCounter(mValue)
		}
		if err != nil {
			http.Error(rw, "invalid metric value", http.StatusBadRequest)
		} else {
			next.ServeHTTP(rw, r)
		}
	})
}

func run() error {
	srvConf, _ := newConfig("localhost", "8080")
	log.Printf("Listening at %s\n", srvConf.fullAddr())
	return http.ListenAndServe(srvConf.fullAddr(), metricRouter())
}

func metricRouter() chi.Router {
	router := chi.NewRouter()
	router.Use(checkMethod)
	router.Use(checkContentType)
	router.Get("/", rootHandler)
	router.Route("/update", func(router chi.Router) {
		router.Route("/{mType}", func(router chi.Router) {
			router.Use(checkMetricType)
			router.Route("/{mName}", func(router chi.Router) {
				router.Use(checkMetricName)
				router.Post("/", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
					http.Error(rw, "incorrect metric value", http.StatusBadRequest)
				}))
				router.Route("/{mValue}", func(router chi.Router) {
					router.Use(checkMetricValue)
					router.Post("/", updateHandler)
				})
			})
		})
	})
	router.Route("/value", func(router chi.Router) {
		router.Route("/{mType}", func(router chi.Router) {
			router.Use(checkMetricType)
			router.Route("/{mName}", func(router chi.Router) {
				router.Use(checkMetricName)
				router.Get("/", valueHandler)
			})
		})
	})
	return router
}

func checkTypeGauge(value string) (gauge, error) {
	converted, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("wrong format %w", err)
	}
	return gauge(converted), nil
}
func checkTypeCounter(value string) (counter, error) {
	converted, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("wrong format %w", err)
	}
	return counter(converted), nil
}
