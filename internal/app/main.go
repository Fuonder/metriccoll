package app

import (
	"context"
	"github.com/Fuonder/metriccoll.git/internal/grpc/service"
	"github.com/Fuonder/metriccoll.git/internal/logger"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"net/http"
)

type Application struct {
	srv  *http.Server
	grpc *service.Service
}

func NewApplication(srv *http.Server, grpc *service.Service) *Application {
	return &Application{
		srv:  srv,
		grpc: grpc,
	}
}

func (a *Application) Run(ctx context.Context) error {
	group, ctx := errgroup.WithContext(ctx)

	group.Go(func() error {
		logger.Log.Info("Starting HTTP server", zap.String("addr", a.srv.Addr))
		err := a.srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.Log.Error("HTTP server error", zap.Error(err))
			return err
		}
		return nil
	})

	group.Go(func() error {
		logger.Log.Info("Starting gRPC server")
		errCh := make(chan error, 1)
		go func() {
			errCh <- a.grpc.Run()
		}()

		select {
		case <-ctx.Done():
			// ctx canceled: signal to shutdown
			return a.grpc.Close()
		case err := <-errCh:
			return err
		}
	})

	// Wait for shutdown signal or error
	if err := group.Wait(); err != nil {
		logger.Log.Info("Shutting down due to error", zap.Error(err))
	} else {
		logger.Log.Info("Shutting down gracefully")
	}

	return nil
}

func (a *Application) Close(ctx context.Context) error {
	var group errgroup.Group

	group.Go(func() error {
		return a.srv.Shutdown(ctx)
	})

	group.Go(func() error {
		return a.grpc.Close()
	})

	if err := group.Wait(); err != nil {
		logger.Log.Error("Error during shutdown", zap.Error(err))
		return err
	}
	logger.Log.Info("Application shutdown complete")
	return nil
}
