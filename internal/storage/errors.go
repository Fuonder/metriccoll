package storage

import "errors"

var ErrFieldNotFound = errors.New("field not found")
var ErrMetricNotFound = errors.New("metric with such key is not found")
