// Package models содержит типы и структуры данных, используемые для представления метрик,
// таких как Gauge и Counter, а также функции для их валидации и обработки.
// Пакет предоставляет возможность работать с метриками различного типа и конвертировать их
// из строковых значений в соответствующие типы с валидацией формата.
package models

import (
	"fmt"
	"strconv"
)

// Int64Ptr — функция (макрос), которая принимает int64 и возвращает ссылку на него.
// Данная функция необходима для упрощения работы со структурой Metrics
func Int64Ptr(i int64) *int64 { return &i }

// Float64Ptr — функция (макрос), которая принимает float64 и возвращает ссылку на него.
// Данная функция необходима для упрощения работы со структурой Metrics
func Float64Ptr(f float64) *float64 { return &f }

// Gauge — тип для представления метрик с плавающей точкой, используется для значений,
// которые могут увеличиваться и уменьшаться.
type Gauge float64

// Counter — тип для представления метрик с целочисленными значениями,
// используется для счетчиков, которые только увеличиваются.
type Counter int64

// CheckTypeGauge преобразует строковое значение в тип Gauge.
// Если строка не может быть преобразована в число с плавающей точкой,
// возвращает ошибку.
func CheckTypeGauge(value string) (Gauge, error) {
	converted, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("wrong format %w", err)
	}
	return Gauge(converted), nil
}

// CheckTypeCounter преобразует строковое значение в тип Counter.
// Если строка не может быть преобразована в целое число, возвращает ошибку.
func CheckTypeCounter(value string) (Counter, error) {
	converted, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("wrong format %w", err)
	}
	return Counter(converted), nil
}

// Type возвращает строковое представление типа для метрики Gauge, которое всегда будет "gauge".
func (t Gauge) Type() string {
	return "gauge"
}

// Type возвращает строковое представление типа для метрики Counter, которое всегда будет "counter".
func (t Counter) Type() string {
	return "counter"
}

// Metrics представляет метрики с идентификатором, типом и значениями.
// В зависимости от типа метрики, значение может быть указано в Metrics.delta для Counter или Metrics.value для Gauge.
type Metrics struct {
	// Идентификатор метрики
	ID string `json:"id"`
	// Тип метрики, может быть "counter" или "gauge"
	MType string `json:"type"`
	// Значение метрики типа Counter
	Delta *int64 `json:"delta,omitempty"`
	// Значение метрики типа Gauge
	Value *float64 `json:"value,omitempty"`
}
