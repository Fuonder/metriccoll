package main

import (
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
)

func TestAppendGaugeMetric(t *testing.T) {
	type data struct {
		st    MemStorage
		items map[string]gauge
	}
	tests := []struct {
		name string
		data data
		want gauge
	}{
		{
			name: "PositiveTestRewrite",
			data: data{
				st: MemStorage{
					gMetric: map[string]gauge{"gMetric1": 3.00},
					cMetric: nil,
					mu:      sync.Mutex{},
				},
				items: map[string]gauge{"gMetric1": 1.00},
			},
			want: 1.00,
		},
		{
			name: "PositiveTestWrite",
			data: data{
				st: MemStorage{
					gMetric: make(map[string]gauge),
					cMetric: nil,
					mu:      sync.Mutex{},
				},
				items: map[string]gauge{"gMetric1": -1.00},
			},
			want: -1.00,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			st := &test.data.st
			require.NotNil(t, &st)
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
		st    MemStorage
		items map[string]counter
	}
	tests := []struct {
		name string
		data data
		want counter
	}{
		{
			name: "PositiveTestIncrease",
			data: data{
				st: MemStorage{
					gMetric: nil,
					cMetric: map[string]counter{"cMetric1": 3},
					mu:      sync.Mutex{},
				},
				items: map[string]counter{"cMetric1": 1},
			},
			want: 4,
		},
		{
			name: "PositiveTestWriteFirst",
			data: data{
				st: MemStorage{
					gMetric: nil,
					cMetric: make(map[string]counter),
					mu:      sync.Mutex{},
				},
				items: map[string]counter{"cMetric1": 1},
			},
			want: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			st := &test.data.st
			require.NotNil(t, &st)
			for name, value := range test.data.items {
				st.AppendCounterMetric(name, value)
				require.Contains(t, st.cMetric, name)
				require.Equal(t, test.want, st.cMetric[name])
			}
		})
	}
}
