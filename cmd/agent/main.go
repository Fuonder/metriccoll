package main

import (
	"context"
	"errors"
	"github.com/Fuonder/metriccoll.git/internal/logger"
	"github.com/Fuonder/metriccoll.git/internal/metrics/MemoryCollector"
	"github.com/Fuonder/metriccoll.git/internal/storage/agentCollection"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"log"
)

var (
	ErrCouldNotSendRequest = errors.New("could not send request")
	ErrWrongResponseStatus = errors.New("wrong request data or metrics value")
)

func main() {
	if err := logger.Initialize("Info"); err != nil {
		panic(err)
	}

	logger.Log.Info("Starting agent")

	mc, err := agentcollection.NewMetricsCollection()
	if err != nil {
		log.Fatal(err)
	}

	err = parseFlags()
	if err != nil {
		log.Fatal(err)
	}
	logger.Log.Info("parse flags success")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := new(errgroup.Group)

	jobsCh := make(chan []byte, 10)
	defer close(jobsCh)

	timeIntervals := memcollector.NewTimeIntervals(CliOpt.ReportInterval, CliOpt.PollInterval)
	collector := memcollector.NewMemoryCollector(mc, timeIntervals, jobsCh)

	err = collector.SetRemoteIP(CliOpt.NetAddr.String())
	if err != nil {
		logger.Log.Fatal("", zap.Error(err))
	}

	err = collector.SetHashKey(CliOpt.HashKey)
	if err != nil {
		logger.Log.Fatal("", zap.Error(err))
	}

	g.Go(func() error {
		err = collector.Collect(ctx, cancel)
		if err != nil {
			return err
		}
		return nil
	})

	g.Go(func() error {
		err = collector.RunWorkers(CliOpt.RateLimit)
		if err != nil {
			cancel()
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Log.Debug("agent exited with error", zap.Error(err))
		cancel()
		panic(err)
	}

	//err = testAll()
	//if err != nil {
	//	close(ch)
	//	time.Sleep(2 * time.Second)
	//	log.Fatal(err)
	//}
	//}
}
