package storage

import (
	"testing"

	model "github.com/Fuonder/metriccoll.git/internal/models"
	"github.com/stretchr/testify/require"
)

func TestAppendGaugeMetric(t *testing.T) {
	type data struct {
		initItem map[string]model.Gauge
		items    map[string]model.Gauge
	}
	tests := []struct {
		name string
		data data
		want model.Gauge
	}{
		{
			name: "PositiveTestRewrite",
			data: data{
				initItem: map[string]model.Gauge{"gMetric1": 3.00},
				items:    map[string]model.Gauge{"gMetric1": 1.00},
			},
			want: 1.00,
		},
		{
			name: "PositiveTestWrite",
			data: data{
				initItem: nil,
				items:    map[string]model.Gauge{"gMetric1": -1.00},
			},
			want: -1.00,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			st, err := NewMemStorage()
			require.NoError(t, err)
			require.NotNil(t, st)
			if test.data.initItem != nil {
				for name, value := range test.data.initItem {
					st.appendGaugeMetric(name, value)
				}
			}
			for name, value := range test.data.items {
				st.appendGaugeMetric(name, value)
				require.Contains(t, st.gMetric, name)
				require.Equal(t, test.want, st.gMetric[name])
			}
		})
	}
}

func TestAppendCounterMetric(t *testing.T) {
	type data struct {
		initItem map[string]model.Counter
		items    map[string]model.Counter
	}
	tests := []struct {
		name string
		data data
		want model.Counter
	}{
		{
			name: "PositiveTestIncrease",
			data: data{
				initItem: map[string]model.Counter{"cMetric1": 3},
				items:    map[string]model.Counter{"cMetric1": 1},
			},
			want: 4,
		},
		{
			name: "PositiveTestWriteFirst",
			data: data{
				initItem: nil,
				items:    map[string]model.Counter{"cMetric1": 1},
			},
			want: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			st, err := NewMemStorage()
			require.NoError(t, err)
			require.NotNil(t, st)
			if test.data.initItem != nil {
				for name, value := range test.data.initItem {
					st.appendCounterMetric(name, value)
				}
			}
			for name, value := range test.data.items {
				st.appendCounterMetric(name, value)
				require.Contains(t, st.cMetric, name)
				require.Equal(t, test.want, st.cMetric[name])
			}
		})
	}
}
