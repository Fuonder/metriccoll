package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestMetrics_updateValues(t *testing.T) {
	tests := []struct {
		name string
		want counter
	}{
		{
			name: "PositiveTest",
			want: 5,
		},
	}
	for _, test := range tests {
		collection, err := NewMetricsCollection()
		require.NoError(t, err)
		require.NotNil(t, collection)
		collection.updateValues(pollInterval)
		time.Sleep(reportInterval + 1*time.Second)
		result, err := collection.getPollCount()
		require.NoError(t, err)
		collection.mu.Lock()
		assert.Equal(t, test.want, result)
		collection.mu.Unlock()

	}
}
