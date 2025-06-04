package memcollector

import "time"

type TimeIntervals struct {
	reportInterval time.Duration
	pollInterval   time.Duration
}

func NewTimeIntervals(rInterval time.Duration, pInterval time.Duration) *TimeIntervals {
	return &TimeIntervals{
		reportInterval: rInterval,
		pollInterval:   pInterval,
	}
}
