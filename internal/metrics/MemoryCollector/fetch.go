package memcollector

import (
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/models"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

func getMemoryInfo() ([]models.Metrics, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return []models.Metrics{}, err
	}

	var mList []models.Metrics

	totalMemoryFloat := float64(v.Total)
	freeMemoryFloat := float64(v.Available)
	mList = append(mList, models.Metrics{
		ID:    "TotalMemory",
		MType: "gauge",
		Delta: nil,
		Value: &totalMemoryFloat,
	})
	mList = append(mList, models.Metrics{
		ID:    "FreeMemory",
		MType: "gauge",
		Delta: nil,
		Value: &freeMemoryFloat,
	})

	return mList, nil

}

func getCPUUtilization() ([]models.Metrics, error) {
	var mList []models.Metrics
	percentages, _ := cpu.Percent(0, true)

	for idx, p := range percentages {
		mt := models.Metrics{
			ID:    fmt.Sprintf("CPUutilization%d", idx),
			MType: "gauge",
			Delta: nil,
			Value: &p,
		}
		mList = append(mList, mt)
	}
	return mList, nil
}
