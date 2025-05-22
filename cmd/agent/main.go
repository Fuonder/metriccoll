package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/buildinfo"
	"github.com/Fuonder/metriccoll.git/internal/certmanager"
	"github.com/Fuonder/metriccoll.git/internal/logger"
	memcollector "github.com/Fuonder/metriccoll.git/internal/metrics/MemoryCollector"
	agentcollection "github.com/Fuonder/metriccoll.git/internal/storage/agentCollection"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"os"
	"os/signal"
	"syscall"
)

var (
	ErrCouldNotSendRequest = errors.New("could not send request")
	ErrWrongResponseStatus = errors.New("wrong request data or metrics value")
)

//go:generate go run ../generator/buildinfo/genBuildInfo.go

func main() {
	bInfo := buildinfo.NewBuildInfo(buildVersion, buildCommit, buildDate, GeneratedBuildInfo)
	fmt.Println(bInfo.String())

	if err := logger.Initialize("Debug"); err != nil {
		panic(err)
	}

	logger.Log.Info("Starting agent")
	err := parseFlags()
	if err != nil {
		logger.Log.Fatal("error during parsing flags: ", zap.Error(err))
	}
	logger.Log.Debug("Flags parsed",
		zap.String("flags", CliOpt.String()))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	mc, err := agentcollection.NewMetricsCollection()
	if err != nil {
		logger.Log.Fatal("can not create collection:", zap.Error(err))
	}

	g := new(errgroup.Group)

	jobsCh := make(chan []byte, 10)

	timeIntervals := memcollector.NewTimeIntervals(CliOpt.ReportInterval, CliOpt.PollInterval)
	cipherManger, err := certmanager.NewCertManager()
	if err != nil {
		logger.Log.Fatal("can not create cert manager", zap.Error(err))
	}
	err = cipherManger.LoadCertificate(CliOpt.CryptoKey)
	if err != nil {
		logger.Log.Fatal("can not load certificate", zap.Error(err))
	}

	collector := memcollector.NewMemoryCollector(mc, timeIntervals, jobsCh, cipherManger)

	err = collector.SetRemoteIP(CliOpt.NetAddr.String())
	if err != nil {
		logger.Log.Fatal("", zap.Error(err))
	}

	err = collector.SetHashKey(CliOpt.HashKey)
	if err != nil {
		logger.Log.Fatal("", zap.Error(err))
	}

	g.Go(func() error {
		err := collector.Collect(ctx, cancel)
		close(jobsCh)
		return err
	})

	g.Go(func() error {
		return collector.RunWorkers(CliOpt.RateLimit)
	})

	g.Go(func() error {
		select {
		case sig := <-sigCh:
			logger.Log.Info("got sigterm", zap.String("signal", sig.String()))
			cancel()
		case <-ctx.Done():
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
	logger.Log.Info("agent exited gracefully")
}
