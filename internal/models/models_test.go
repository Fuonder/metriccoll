package models

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCheckTypeGauge(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    Gauge
		wantErr bool
	}{
		{
			name:    "PositiveGauge",
			value:   "123.45",
			want:    Gauge(123.45),
			wantErr: false,
		},
		{
			name:    "NegativeGaugeInvalidFormat",
			value:   "invalid",
			want:    0,
			wantErr: true,
		},
		{
			name:    "ZeroGauge",
			value:   "0",
			want:    Gauge(0),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CheckTypeGauge(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				require.Equal(t, tt.want, result)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, result)
			}
		})
	}
}

func TestCheckTypeCounter(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    Counter
		wantErr bool
	}{
		{
			name:    "PositiveCounter",
			value:   "12345",
			want:    Counter(12345),
			wantErr: false,
		},
		{
			name:    "NegativeCounterInvalidFormat",
			value:   "invalid",
			want:    0,
			wantErr: true,
		},
		{
			name:    "ZeroCounter",
			value:   "0",
			want:    Counter(0),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CheckTypeCounter(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				require.Equal(t, tt.want, result)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, result)
			}
		})
	}
}

func TestGaugeTypeMethod(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "GaugeType",
			want: "gauge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var g Gauge
			require.Equal(t, tt.want, g.Type())
		})
	}
}

func TestCounterTypeMethod(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "CounterType",
			want: "counter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c Counter
			require.Equal(t, tt.want, c.Type())
		})
	}
}

func TestMetricsInitialization(t *testing.T) {
	tests := []struct {
		name    string
		metrics Metrics
	}{
		{
			name: "PositiveMetricsInitializationGauge",
			metrics: Metrics{
				ID:    "metric1",
				MType: "gauge",
				Value: new(float64),
			},
		},
		{
			name: "PositiveMetricsInitializationCounter",
			metrics: Metrics{
				ID:    "metric2",
				MType: "counter",
				Delta: new(int64),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metrics.MType == "gauge" {
				*tt.metrics.Value = 100.5
				require.Equal(t, "metric1", tt.metrics.ID)
				require.Equal(t, "gauge", tt.metrics.MType)
				require.Equal(t, 100.5, *tt.metrics.Value)
			} else if tt.metrics.MType == "counter" {
				*tt.metrics.Delta = 10
				require.Equal(t, "metric2", tt.metrics.ID)
				require.Equal(t, "counter", tt.metrics.MType)
				require.Equal(t, int64(10), *tt.metrics.Delta)
			}
		})
	}
}
