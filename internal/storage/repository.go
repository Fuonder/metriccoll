// Package storage содержит интерфейсы для взаимодействия с хранилищем данных,
// как с файловыми системами, так и с базами данных. Эти интерфейсы описывают
// работу с метриками: операции чтения и записи, а также управление соединениями.
package storage

import (
	"context"
	"time"

	"github.com/Fuonder/metriccoll.git/internal/models"
)

// MetricReader интерфейс для чтения метрик.
// Позволяет получить все метрики или метрику по имени и типу.
type MetricReader interface {
	// GetAllMetrics возвращает все метрики.
	GetAllMetrics() []models.Metrics
	// GetMetricByName возвращает метрику по имени и типу.
	GetMetricByName(name string, mType string) (models.Metrics, error)
}

// MetricWriter интерфейс для записи метрик.
// Позволяет добавлять одну или несколько метрик.
type MetricWriter interface {
	// AppendMetric добавляет одну метрику.
	AppendMetric(metric models.Metrics) error
	// AppendMetrics добавляет несколько метрик.
	AppendMetrics([]models.Metrics) error
}

// MetricFileHandler интерфейс для работы с файлами.
// Позволяет выгружать метрики в файл или загружать метрики из файла.
type MetricFileHandler interface {
	// DumpMetrics выгружает метрики в файл.
	DumpMetrics() error
	// loadMetricsFromFile загружает метрики из файла.
	loadMetricsFromFile() error
}

// MetricDatabaseHandler интерфейс для проверки соединения с базой данных.
type MetricDatabaseHandler interface {
	// CheckConnection проверяет подключение к базе данных.
	CheckConnection() error
}

// Collection интерфейс для работы с коллекцией метрик.
// Предназначен для чтения значений и обновления их с заданным интервалом.
type Collection interface {
	// ReadValues читает значения метрик.
	ReadValues()
	// UpdateValues обновляет значения метрик через заданные интервалы.
	UpdateValues(ctx context.Context, interval time.Duration)
	// GetCounterList возвращает список всех метрик типа Counter.
	GetCounterList() map[string]models.Counter
	// GetGaugeList возвращает список всех метрик типа Gauge.
	GetGaugeList() map[string]models.Gauge
}

// DBConnection интерфейс для работы с базой данных.
// Включает операции для чтения, записи, создания таблиц и проверки соединения.
type DBConnection interface {
	DBReader
	DBWriter
	// CreateTablesContext создает таблицы в базе данных в контексте.
	CreateTablesContext(ctx context.Context) error
	// TryConnectContext пытается подключиться к базе данных в контексте.
	TryConnectContext(ctx context.Context) error
	// Close закрывает соединение с базой данных.
	Close() error
}

// DBReader интерфейс для чтения метрик из базы данных.
type DBReader interface {
	// GetGaugeMetric получает метрику типа Gauge по имени.
	GetGaugeMetric(ctx context.Context, name string) (models.Metrics, error)
	// GetCounterMetric получает метрику типа Counter по имени.
	GetCounterMetric(ctx context.Context, name string) (models.Metrics, error)
	// GetAllMetrics получает все метрики из базы данных.
	GetAllMetrics(ctx context.Context) ([]models.Metrics, error)
}

// DBWriter интерфейс для записи метрик в базу данных.
type DBWriter interface {
	// AppendGaugeMetric добавляет метрику типа Gauge в базу данных.
	AppendGaugeMetric(ctx context.Context, metric models.Metrics) error
	// AppendCounterMetric добавляет метрику типа Counter в базу данных.
	AppendCounterMetric(ctx context.Context, metric models.Metrics) error
	// AppendBatch добавляет несколько метрик в базу данных.
	AppendBatch(ctx context.Context, metrics []models.Metrics) error
}
