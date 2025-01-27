package storage

import (
	"encoding/json"
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/models"
	"io"
	"os"
	"sync"
	"time"
)

type StoreMode struct {
	Sync          bool
	StoreInterval time.Duration
}

type JSONStorage struct {
	metrics      []models.Metrics
	fStore       bool
	fStoragePath string
	Mode         StoreMode
	mu           sync.Mutex
}

func NewJSONStorage(loadFromFile bool, filePath string, interval time.Duration) (*JSONStorage, error) {
	st := JSONStorage{metrics: make([]models.Metrics, 0)}
	st.fStoragePath = filePath
	st.fStore = loadFromFile
	if st.fStore {
		err := st.loadMetricsFromFile()
		if err != nil {
			return nil, err
		}
	}
	if interval == 0 {
		st.Mode.Sync = true
	} else {
		st.Mode.Sync = false
		st.Mode.StoreInterval = interval
	}
	return &st, nil
}

func (st *JSONStorage) DumpMetrics() error {

	data, err := json.MarshalIndent(st.metrics, "", "    ")
	if err != nil {
		return err
	}
	err = os.WriteFile(st.fStoragePath, data, 0666)
	if err != nil {
		return err
	}
	//for _, m := range st.metrics {
	//	data, err := json.MarshalIndent(m, "", "    ")
	//	if err != nil {
	//		return err
	//	}
	//	err = os.WriteFile(st.fStoragePath, data, 0666)
	//	if err != nil {
	//		return err
	//	}
	//}
	return nil
}

func (st *JSONStorage) loadMetricsFromFile() error {
	_, err := os.Stat(st.fStoragePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("can not find metrcis file \"%s\": %w", st.fStoragePath, err)
	} else if err != nil {
		return fmt.Errorf("can not find metrcis file \"%s\": %w", st.fStoragePath, err)
	}
	file, err := os.Open(st.fStoragePath)
	if err != nil {
		return fmt.Errorf("can not open metrcis file \"%s\": %w", st.fStoragePath, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	var items []models.Metrics
	err = json.Unmarshal(data, &items)
	if err != nil {
		return err
	}
	st.metrics = items

	//decoder := json.NewDecoder(file)

	//for {
	//	var m models.Metrics
	//	if err := decoder.Decode(&m); err != nil {
	//		if err == io.EOF {
	//			break
	//		}
	//		return fmt.Errorf("can not decode metrcis file \"%s\": %w", st.fStoragePath, err)
	//	}
	//	err = st.AppendMetric(m)
	//	if err != nil {
	//		return fmt.Errorf("can not load metric %s from \"%s\": %w", m.ID, st.fStoragePath, err)
	//	}
	//}
	return nil
}

func (st *JSONStorage) AppendMetric(metric models.Metrics) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	for i, existingItem := range st.metrics {
		if existingItem.ID == metric.ID && existingItem.MType == metric.MType {
			if metric.MType == "gauge" {
				if metric.Value == nil {
					return ErrInvalidMetricValue
				}
				*st.metrics[i].Value = *metric.Value
				if st.Mode.Sync {
					err := st.DumpMetrics()
					if err != nil {
						return err
					}
				}
				return nil
			} else if metric.MType == "counter" {
				if metric.Delta == nil {
					return ErrInvalidMetricValue
				}
				*st.metrics[i].Delta += *metric.Delta
				if st.Mode.Sync {
					err := st.DumpMetrics()
					if err != nil {
						return err
					}
				}
				return nil
			} else {
				return fmt.Errorf("metric type: %s is not supported", metric.MType)
			}
		}
	}
	if metric.MType == "gauge" {
		if metric.Value == nil {
			return ErrInvalidMetricValue
		}
		st.metrics = append(st.metrics, metric)
		if st.Mode.Sync {
			err := st.DumpMetrics()
			if err != nil {
				return err
			}
		}
		return nil
	} else if metric.MType == "counter" {
		if metric.Delta == nil {
			return ErrInvalidMetricValue
		}
		st.metrics = append(st.metrics, metric)
		if st.Mode.Sync {
			err := st.DumpMetrics()
			if err != nil {
				return err
			}
		}
		return nil
	} else {
		return fmt.Errorf("metric type: %s is not supported", metric.MType)
	}
}

func (st *JSONStorage) GetMetricByName(name string, mType string) (models.Metrics, error) {
	for i, existingItem := range st.metrics {
		if existingItem.ID == name && existingItem.MType == mType {
			return st.metrics[i], nil
		}
	}
	return models.Metrics{}, fmt.Errorf("%v: %s", ErrMetricNotFound, name)
}

func (st *JSONStorage) GetAllMetrics() []models.Metrics {
	return st.metrics
}
