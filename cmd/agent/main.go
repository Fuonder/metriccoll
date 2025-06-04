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
	pb "github.com/Fuonder/metriccoll.git/proto"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	var (
		service memcollector.Collector
		err     error
		useGRPC bool = false
		conn    *grpc.ClientConn
	)

	if CliOpt.GRPCAddress != "" {
		useGRPC = true
	}

	service, conn, err = prepareService(CliOpt, jobsCh, useGRPC)
	if err != nil {
		close(jobsCh)
		return err
	}
	defer func() {
		if conn != nil {
			conn.Close()
			logger.Log.Info("gRPC connection closed")
		}
	}()

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

func prepareService(CliOpt *CliOptions, jobsCh chan []byte, useGRPC bool) (collector memcollector.Collector, conn *grpc.ClientConn, err error) {
	mc, err := agentcollection.NewMetricsCollection()
	if err != nil {
		logger.Log.Info("can not create collection:", zap.Error(err))
		return nil, nil, err
	}

	timeIntervals := memcollector.NewTimeIntervals(CliOpt.ReportInterval, CliOpt.PollInterval)
	cipherManger, err := certmanager.NewCertManager()
	if err != nil {
		logger.Log.Info("can not create cert manager", zap.Error(err))
		return nil, nil, err
	}
	err = cipherManger.LoadCertificate(CliOpt.CryptoKey)
	if err != nil {
		logger.Log.Info("can not load certificate", zap.Error(err))
		return nil, nil, err
	}

	if !useGRPC {
		collector, err = memcollector.NewMemoryCollector(mc, timeIntervals, jobsCh, cipherManger)
		if err != nil {
			logger.Log.Fatal("", zap.Error(err))
		}
		err = collector.(*memcollector.MemoryCollector).SetRemoteIP(CliOpt.NetAddr.String())
		if err != nil {
			logger.Log.Info("Can not set Remote IP address", zap.Error(err))
			return nil, nil, err
		}
	} else {
		conn, err = grpc.NewClient(CliOpt.GRPCAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			logger.Log.Info("Cannot create grpc client", zap.Error(err))
			return nil, nil, err
		}

		c := pb.NewMetricsClient(conn)
		collector, err = memcollector.NewMemoryGRPCCollector(mc, timeIntervals, jobsCh, cipherManger, c)
		if err != nil {
			conn.Close()
			logger.Log.Fatal("", zap.Error(err))
		}
	}

	err = collector.SetHashKey(CliOpt.HashKey)
	if err != nil {
		logger.Log.Info("Can not set Remote Hash Key", zap.Error(err))
		return nil, nil, err
	}

	return collector, conn, nil
}
