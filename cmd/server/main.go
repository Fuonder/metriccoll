package main

import (
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/logger"
	"github.com/Fuonder/metriccoll.git/internal/server"
	"github.com/Fuonder/metriccoll.git/internal/storage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"log"
	"net/http"
	"time"
)

func main() {
	err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}
	if err := logger.Initialize(FlagsOptions.LogLevel); err != nil {
		panic(fmt.Errorf("method run: %v", err))
	}
	logger.Log.Debug("Flags parsed",
		zap.String("flags", FlagsOptions.String()))

	logger.Log.Info("Starting metric collector")
	if err = run(); err != nil {
		logger.Log.Fatal("", zap.Error(err))
	}
}

func createJSONStorage() (storage.Storage, error) {
	settings := storage.NewFileStoreInfo(FlagsOptions.FileStoragePath, FlagsOptions.StoreInterval, FlagsOptions.Restore)
	ms, err := storage.NewJSONStorage(settings)
	if err != nil {
		return &storage.JSONStorage{}, err
	}

	if !ms.FileInfo.Sync {
		go func() {
			for {
				time.Sleep(ms.FileInfo.StoreInterval)
				_ = ms.DumpMetrics()
			}
		}()
	}
	return ms, nil
}

func run() error {

	var handler *server.Handler

	dbSettings := FlagsOptions.DatabaseDSN
	dbStorage, err := storage.NewDatabase(dbSettings)
	if err != nil {
		logger.Log.Warn("Cannot connect to db")
		logger.Log.Info("Switching to file(json) storage")
		jsonStorage, err := createJSONStorage()
		if err != nil {
			return err
		}
		handler = server.NewHandler(jsonStorage)
	} else {
		logger.Log.Info("Connected to db")
		err = dbStorage.CreateTables()
		if err != nil {
			return err
		}
		handler = server.NewHandler(dbStorage)
		defer dbStorage.Close()
	}

	logger.Log.Info("Listening at",
		zap.String("Addr", netAddr.String()))
	return http.ListenAndServe(netAddr.String(), metricRouter(handler))
}

func metricRouter(h *server.Handler) chi.Router {
	logger.Log.Debug("Entering router")
	router := chi.NewRouter()

	router.Use(h.CheckMethod)
	router.Use(h.CheckContentType)
	router.Get("/", logger.HanlderWithLogger(server.GzipMiddleware(h.RootHandler)))
	router.Route("/ping", func(router chi.Router) {
		router.Get("/", logger.HanlderWithLogger(server.GzipMiddleware(h.DBPingHandler)))
	})
	router.Route("/updates", func(router chi.Router) {
		router.Post("/", logger.HanlderWithLogger(server.GzipMiddleware(h.MultipleUpdateHandler)))
	})
	router.Route("/update", func(router chi.Router) {
		router.Post("/", logger.HanlderWithLogger(server.GzipMiddleware(h.JSONUpdateHandler)))
		router.Route("/{mType}", func(router chi.Router) {
			router.Use(h.CheckMetricType)
			router.Route("/{mName}", func(router chi.Router) {
				router.Use(h.CheckMetricName)
				router.Post("/", logger.HanlderWithLogger(server.GzipMiddleware(func(rw http.ResponseWriter, r *http.Request) {
					logger.Log.Debug("no metric value has given")
					http.Error(rw, "incorrect metric value", http.StatusBadRequest)
				})))
				router.Route("/{mValue}", func(router chi.Router) {
					router.Use(h.CheckMetricValue)
					router.Post("/", logger.HanlderWithLogger(server.GzipMiddleware(h.UpdateHandler)))
				})
			})
		})
	})
	router.Route("/value", func(router chi.Router) {
		router.Post("/", logger.HanlderWithLogger(server.GzipMiddleware(h.JSONGetHandler)))
		// router.Post("/", -> JSON VALUE GET HANDLER)
		router.Route("/{mType}", func(router chi.Router) {
			router.Use(h.CheckMetricType)
			router.Route("/{mName}", func(router chi.Router) {
				router.Use(h.CheckMetricName)
				router.Get("/", logger.HanlderWithLogger(server.GzipMiddleware(h.ValueHandler)))
			})
		})
	})
	return router
}

func loadMetrics() {

}
func metricsSave() {}
