package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
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

func run() error {
	srvConf, _ := newConfig("localhost", "8080")
	log.Printf("Listening at %s\n", srvConf.fullAddr())
	return http.ListenAndServe(srvConf.fullAddr(), http.HandlerFunc(metricHandler))
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

func processData(URL *url.URL) int {
	stringURL := URL.String()
	urlParts := strings.Split(stringURL, "/")
	urlPartsLen := len(urlParts)

	log.Printf("URL length %d \n", urlPartsLen)
	log.Println(urlParts)
	if urlParts[1] != "update" || urlPartsLen < 5 {
		return http.StatusNotFound
	}
	if urlParts[3] == "" {
		return http.StatusNotFound
	}

	log.Println("Guessing types")
	if urlParts[2] == "gauge" {
		log.Println("Guessing type \"gauge\"")
		log.Println("Guessing - YES")
		log.Println("Checking value")
		value, err := checkTypeGauge(urlParts[4])
		if err != nil {
			log.Println(err)
			return http.StatusBadRequest
		}
		log.Println("Value - OK")
		log.Println("Appending to storage")
		ms.AppendGaugeMetric(urlParts[3], value)
	} else if urlParts[2] == "counter" {
		log.Println("Guessing type \"counter\"")
		log.Println("Guessing - YES")
		log.Println("Checking value")
		value, err := checkTypeCounter(urlParts[4])
		if err != nil {
			log.Println(err)
			return http.StatusBadRequest
		}
		log.Println("Value - OK")
		log.Println("Appending to storage")
		ms.AppendCounterMetric(urlParts[3], value)
	} else {
		log.Println("Unsupported type")
		return http.StatusBadRequest
	}
	log.Println("Gauge: ", ms.gMetric)
	log.Println("Counter: ", ms.cMetric)
	return http.StatusOK
}

func metricHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Got request")
	log.Println("Validating request")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	log.Println("Method - OK")
	if r.Header.Get("Content-Type") != "text/plain" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Println("Content-Type - OK")
	log.Println("Processing URL")
	err := processData(r.URL)
	w.WriteHeader(err)
}
