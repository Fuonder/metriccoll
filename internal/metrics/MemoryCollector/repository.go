package memcollector

import (
	"context"
	"github.com/Fuonder/metriccoll.git/internal/storage"
)

type Collector interface {
	SetStorage(stArg storage.Collection) error
	SetHashKey(key string) error
	Collect(ctx context.Context, cancel context.CancelFunc) error
	RunWorkers(rateLimit int64) error
	WaitWorkers()
}
