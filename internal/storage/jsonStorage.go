package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/Fuonder/metriccoll.git/internal/logger"
	"github.com/Fuonder/metriccoll.git/internal/models"
	"go.uber.org/zap"
)

type FileStoreInfo struct {
	Sync          bool
	StoreInterval time.Duration
	fLoadFromFile bool
	fPath         string
}

func NewFileStoreInfo(fPath string, interval time.Duration, fLoadFromFile bool) *FileStoreInfo {
	syncMode := false
	if interval == 0 {
		syncMode = true
	}

	return &FileStoreInfo{
		Sync:          syncMode,
		StoreInterval: interval,
		fLoadFromFile: fLoadFromFile,
		fPath:         fPath,
	}
}

type JSONStorage struct {
	metrics  []models.Metrics
	fileInfo *FileStoreInfo
	mu       sync.RWMutex
	fileMu   sync.RWMutex
}

func NewJSONStorage(fileStoreInfo *FileStoreInfo) (*JSONStorage, error) {

	st := JSONStorage{metrics: make([]models.Metrics, 0), fileInfo: fileStoreInfo}

	if st.fileInfo.fLoadFromFile {
		err := st.loadMetricsFromFile()
		if err != nil {
			return nil, err
		}
	}
	return &st, nil
}

func (st *JSONStorage) IsSyncFileMode() bool {
	return st.fileInfo.Sync
}

func (st *JSONStorage) DumpMetrics() error {
	st.fileMu.Lock()
	defer st.fileMu.Unlock()
	data, err := json.MarshalIndent(st.metrics, "", "    ")
	if err != nil {
		return err
	}
	err = os.WriteFile(st.fileInfo.fPath, data, OsAllRw)
	if err != nil {
		return err
	}
	return nil
}

func (st *JSONStorage) loadMetricsFromFile() error {
	statTest, err := os.Stat(st.fileInfo.fPath)
	if os.IsNotExist(err) {
		logger.Log.Info("can not find metrcis file",
			zap.String("Expected file", st.fileInfo.fPath),
			zap.Error(err))
		return nil
	} else if err != nil {
		return fmt.Errorf("can not find metrcis file \"%s\": %w", st.fileInfo.fPath, err)
	}
	if statTest.Size() == 0 {
		return nil
	}
	file, err := os.OpenFile(st.fileInfo.fPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	var items []models.Metrics
	err = json.Unmarshal(data, &items)
	if err != nil {
		return fmt.Errorf("not valid json data in file: %w", err)
	}
	st.metrics = items
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
				if st.fileInfo.Sync {
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
				if st.fileInfo.Sync {
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
		if st.fileInfo.Sync {
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
		if st.fileInfo.Sync {
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

func (st *JSONStorage) AppendMetrics(metrics []models.Metrics) error {
	for _, metric := range metrics {
		err := st.AppendMetric(metric)
		if err != nil {
			return err
		}
	}
	return nil
}
