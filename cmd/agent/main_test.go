package main

import (
	"context"
	"os"
	"testing"
	"time"

	model "github.com/Fuonder/metriccoll.git/internal/models"
	agentcollection "github.com/Fuonder/metriccoll.git/internal/storage/agentCollection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				number:  6,
			},
		},
		{
			name: "NegativeTest",
			want: want{
				wantErr: true,
			},
		},
	}
	err := os.Setenv("CRYPTO_KEY", "../../certs/server.crt")
	require.NoError(t, err)
	err = parseFlags()
	require.NoError(t, err)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			collection, err := agentcollection.NewMetricsCollection()
			require.NoError(t, err)
			require.NotNil(t, collection)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			collection.UpdateValues(ctx, CliOpt.PollInterval)
			time.Sleep(CliOpt.ReportInterval)
			cancel()

			result, err := collection.GetPollCount()
			require.NoError(t, err)
			if !test.want.wantErr {
				assert.GreaterOrEqual(t, test.want.number, result)
			} else {
				assert.NotEqual(t, test.want.number, result)
			}
		})
	}
}
