package main

import (
	model "github.com/Fuonder/metriccoll.git/internal/models"
	"github.com/Fuonder/metriccoll.git/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestMetrics_updateValues(t *testing.T) {
	type want struct {
		wantErr bool
		number  model.Counter
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
			collection, err := storage.NewMetricsCollection()
			require.NoError(t, err)
			require.NotNil(t, collection)
			ch := make(chan struct{})
			collection.UpdateValues(CliOpt.PollInterval, ch)
			time.Sleep(CliOpt.ReportInterval)
			close(ch)
			result, err := collection.GetPollCount()
			require.NoError(t, err)
			if !test.want.wantErr {
				assert.Equal(t, test.want.number, result)
			} else {
				assert.NotEqual(t, test.want.number, result)
			}
		})
	}
}
