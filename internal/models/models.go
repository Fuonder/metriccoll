package models

import (
	"fmt"
	"strconv"
)

type Gauge float64
type Counter int64

func CheckTypeGauge(value string) (Gauge, error) {
	converted, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("wrong format %w", err)
	}
	return Gauge(converted), nil
}
func CheckTypeCounter(value string) (Counter, error) {
	converted, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("wrong format %w", err)
	}
	return Counter(converted), nil
}

func (t Gauge) Type() string {
	return "gauge"
}

func (t Counter) Type() string {
	return "counter"
}
