package metrics

import (
	"context"
	"time"

	"github.com/Fuonder/metriccoll.git/internal/storage"
)

type Collector interface {
	SetStorage(collection storage.Collection) error
	SetRemoteIP(remoteIP string) error
	Collect(ctx context.Context, cancel context.CancelFunc) error
	RunWorkers(rateLimit time.Duration) error
}
type Sender interface {
	SetHashKey(key string) error
	Post(packetBody []byte, remoteURL string) error
	CheckConnection() error
}
