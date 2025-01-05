package main

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAppendGaugeMetric(t *testing.T) {
	type data struct {
		initItem map[string]gauge
		items    map[string]gauge
	}
	tests := []struct {
		name string
		data data
		want gauge
	}{
		{
			name: "PositiveTestRewrite",
			data: data{
				initItem: map[string]gauge{"gMetric1": 3.00},
				items:    map[string]gauge{"gMetric1": 1.00},
			},
			want: 1.00,
		},
		{
			name: "PositiveTestWrite",
			data: data{
				initItem: nil,
				items:    map[string]gauge{"gMetric1": -1.00},
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
					st.AppendGaugeMetric(name, value)
				}
			}
			for name, value := range test.data.items {
				st.AppendGaugeMetric(name, value)
				require.Contains(t, st.gMetric, name)
				require.Equal(t, test.want, st.gMetric[name])
			}
		})
	}
}

func TestAppendCounterMetric(t *testing.T) {
	type data struct {
		initItem map[string]counter
		items    map[string]counter
	}
	tests := []struct {
		name string
		data data
		want counter
	}{
		{
			name: "PositiveTestIncrease",
			data: data{
				initItem: map[string]counter{"cMetric1": 3},
				items:    map[string]counter{"cMetric1": 1},
			},
			want: 4,
		},
		{
			name: "PositiveTestWriteFirst",
			data: data{
				initItem: nil,
				items:    map[string]counter{"cMetric1": 1},
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
					st.AppendCounterMetric(name, value)
				}
			}
			for name, value := range test.data.items {
				st.AppendCounterMetric(name, value)
				require.Contains(t, st.cMetric, name)
				require.Equal(t, test.want, st.cMetric[name])
			}
		})
	}
}
