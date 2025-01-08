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

func valueHandler(rw http.ResponseWriter, r *http.Request) {
	log.Println("entering value handler")
	mType := chi.URLParam(r, "mType")
	mName := chi.URLParam(r, "mName")

	if mType == "gauge" {
		value, err := ms.GetGaugeMetric(mName)
		if err != nil {
			log.Println(err)
			http.Error(rw, err.Error(), http.StatusNotFound)
		} else {
			io.WriteString(rw, strconv.FormatFloat(float64(value), 'f', -1, 64))
			return
		}
	} else if mType == "counter" {
		value, err := ms.GetCounterMetric(mName)
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
func updateHandler(rw http.ResponseWriter, r *http.Request) {
	log.Println("Updating metric")
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
		log.Println("Invalid metric type, can not add metric")
		http.Error(rw, "Internal server error", http.StatusInternalServerError)
	}
}
func rootHandler(rw http.ResponseWriter, r *http.Request) {
	log.Println("Entering root handler")
	rw.Header().Set("Content-Type", "text/html")
	var metricList []string
	log.Println("creating metric list")
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
	log.Printf("final metric list: %v", metricList)
	io.WriteString(rw, strings.Join(metricList, ", "))
}

func checkMethod(next http.Handler) http.Handler {
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
func checkContentType(next http.Handler) http.Handler {
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
func checkMetricType(next http.Handler) http.Handler {
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

func checkMetricName(next http.Handler) http.Handler {
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

func checkMetricValue(next http.Handler) http.Handler {
	log.Println("checking metric value")
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		mType := chi.URLParam(r, "mType")
		mValue := chi.URLParam(r, "mValue")
		var err error
		log.Printf("guessing metric type")
		if mType == "gauge" {
			_, err = checkTypeGauge(mValue)
		} else if mType == "counter" {
			_, err = checkTypeCounter(mValue)
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

func main() {
	err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Starting metric collector")
	if err = run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	log.Printf("Listening at %s\n", netAddr.String())
	return http.ListenAndServe(netAddr.String(), metricRouter())
}

func metricRouter() chi.Router {
	log.Println("Entering router")
	router := chi.NewRouter()
	router.Use(checkMethod)
	router.Use(checkContentType)
	router.Get("/", rootHandler)
	router.Route("/update", func(router chi.Router) {
		router.Route("/{mType}", func(router chi.Router) {
			router.Use(checkMetricType)
			router.Route("/{mName}", func(router chi.Router) {
				router.Use(checkMetricName)
				router.Post("/", func(rw http.ResponseWriter, r *http.Request) {
					log.Println("no metric value has given")
					http.Error(rw, "incorrect metric value", http.StatusBadRequest)
				})
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
