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

	if err := run(&CliOpt); err != nil {
		logger.Log.Fatal("error during run", zap.Error(err))
	}
	logger.Log.Info("Agent finished")
}

func run(CliOpt *CliOptions) error {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	g := new(errgroup.Group)

	jobsCh := make(chan []byte, 10)

	service, err := prepareService(CliOpt, jobsCh)
	if err != nil {
		close(jobsCh)
		return err
	}

	g.Go(func() error {
		err := service.Collect(ctx, cancel)
		close(jobsCh)
		return err
	})

	g.Go(func() error {
		return service.RunWorkers(CliOpt.RateLimit)
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
		return err
	}

	logger.Log.Info("agent exited gracefully")
	return nil
}

func prepareService(CliOpt *CliOptions, jobsCh chan []byte) (collector *memcollector.MemoryCollector, err error) {
	mc, err := agentcollection.NewMetricsCollection()
	if err != nil {
		logger.Log.Info("can not create collection:", zap.Error(err))
		return nil, err
	}

	timeIntervals := memcollector.NewTimeIntervals(CliOpt.ReportInterval, CliOpt.PollInterval)
	cipherManger, err := certmanager.NewCertManager()
	if err != nil {
		logger.Log.Info("can not create cert manager", zap.Error(err))
		return nil, err
	}
	err = cipherManger.LoadCertificate(CliOpt.CryptoKey)
	if err != nil {
		logger.Log.Info("can not load certificate", zap.Error(err))
		return nil, err
	}

	collector = memcollector.NewMemoryCollector(mc, timeIntervals, jobsCh, cipherManger)

	err = collector.SetRemoteIP(CliOpt.NetAddr.String())
	if err != nil {
		logger.Log.Info("Can not set Remote IP address", zap.Error(err))
		return nil, err
	}

	err = collector.SetHashKey(CliOpt.HashKey)
	if err != nil {
		logger.Log.Info("Can not set Remote Hash Key", zap.Error(err))
		return nil, err
	}

	return collector, nil
}
