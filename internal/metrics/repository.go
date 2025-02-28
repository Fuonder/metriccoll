package metrics

import (
	"context"
	"github.com/Fuonder/metriccoll.git/internal/storage"
	"time"
)

type Collector interface {
	SetStorage(collection storage.Collection) error
	SetRemoteIP(remoteIP string) error
	Collect(ctx context.Context, cancel context.CancelFunc) error
	RunWorkers(rateLimit time.Duration) error
}
type Sender interface {
	SetHashKey(key string) error
	Post(packetBody []byte, remoteUrl string) error
	CheckConnection() error
}
