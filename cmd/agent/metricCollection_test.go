package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestMetrics_updateValues(t *testing.T) {
	type want struct {
		wantErr bool
		number  counter
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: "PositiveTest",
			want: want{
				wantErr: false,
				number:  5,
			},
		},
		{
			name: "NegativeTest",
			want: want{
				wantErr: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			//mc := MetricsCollection{
			//	gMetrics: make(map[string]gauge),
			//	cMetrics: map[string]counter{"PollCount": 0},
			//}
			//require.NoError(t, err)
			require.NotNil(t, mc)
			for i := 0; i < 5; i++ {
				mc.ReadValues()
				time.Sleep(1 * time.Second)
			}
			//time.Sleep(opt.reportInterval)
			result, err := mc.getPollCount()
			require.NoError(t, err)
			//collection.mu.Lock()
			if !test.want.wantErr {
				assert.Equal(t, test.want.number, result)
			} else {
				assert.NotEqual(t, test.want.number, result)
			}
			//collection.mu.Unlock()
		})
	}
}
