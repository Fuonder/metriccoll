package main

import (
	"context"
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/buildinfo"
	"github.com/Fuonder/metriccoll.git/internal/certmanager"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/Fuonder/metriccoll.git/internal/logger"
	"github.com/Fuonder/metriccoll.git/internal/server"
	"github.com/Fuonder/metriccoll.git/internal/storage"
	"github.com/Fuonder/metriccoll.git/internal/storage/database"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

//go:generate go run ../generator/buildinfo/genBuildInfo.go
//go:generate go run ../generator/certificates/genCertificates.go

func main() {
	bInfo := buildinfo.NewBuildInfo(buildVersion, buildCommit, buildDate, GeneratedBuildInfo)
	fmt.Println(bInfo.String())

	err := parseFlags()
	if err != nil {
		log.Fatalf("error while parsing flags: %v", err)
	}
	if err := logger.Initialize(FlagsOptions.LogLevel); err != nil {
		panic(fmt.Errorf("method run: %v", err))
	}
	logger.Log.Debug("Flags parsed",
		zap.String("flags", FlagsOptions.String()))

	logger.Log.Info("Starting metric MemoryCollector")
	if err = run(); err != nil {
		logger.Log.Fatal("", zap.Error(err))
	}
}

func createJSONStorage() (*storage.JSONStorage, error) {
	settings := storage.NewFileStoreInfo(FlagsOptions.FileStoragePath, FlagsOptions.StoreInterval, FlagsOptions.Restore)
	ms, err := storage.NewJSONStorage(settings)
	if err != nil {
		return &storage.JSONStorage{}, err
	}

	if !ms.IsSyncFileMode() {
		go func() {
			for {
				time.Sleep(FlagsOptions.StoreInterval)
				_ = ms.DumpMetrics()
			}
		}()
	}
	return ms, nil
}

func run() error {
	var handler *server.Handler

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbSettings := FlagsOptions.DatabaseDSN

	cipherManager, err := certmanager.NewCertManager()
	if err != nil {
		return err
	}
	err = cipherManager.LoadPrivateKey(FlagsOptions.CryptoKey)
	if err != nil {
		return err
	}

	dbConnection, err := database.NewPSQLConnection(ctx, dbSettings)
	if err != nil {
		logger.Log.Warn("Cannot connect to db")
		logger.Log.Info("Switching to file(json) storage")
		jsonStorage, err := createJSONStorage()
		if err != nil {
			return err
		}
		handler = server.NewHandler(jsonStorage, jsonStorage, jsonStorage, nil, cipherManager, FlagsOptions.HashKey)
	} else {
		logger.Log.Info("Connected to db")
		err := dbConnection.CreateTablesContext(ctx)
		if err != nil {
			return err
		}

		dbStorage, err := database.NewDBStorage(ctx, dbConnection)
		if err != nil {
			return err
		}
		handler = server.NewHandler(dbStorage, dbStorage, nil, dbStorage, cipherManager, FlagsOptions.HashKey)
		defer func(dbStorage *database.DBStorage) {
			err := dbStorage.Close()
			if err != nil {
				logger.Log.Warn("Cannot close db", zap.Error(err))
			}
		}(dbStorage)
	}

	logger.Log.Info("Listening at",
		zap.String("Addr", FlagsOptions.NetAddress.String()))
	return http.ListenAndServe(FlagsOptions.NetAddress.String(), metricRouter(handler))
}

func metricRouter(h *server.Handler) chi.Router {
	logger.Log.Debug("Entering router")
	router := chi.NewRouter()

	router.Use(h.CheckMethod)
	router.Use(h.CheckContentType)
	router.Use(h.HashMiddleware)
	router.Use(h.DecryptionMiddleware)

	router.Mount("/debug/pprof", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.DefaultServeMux.ServeHTTP(w, r)
	}))

	router.Get("/", logger.HanlderWithLogger(h.WithHashing(server.GzipMiddleware(h.RootHandler))))
	router.Route("/ping", func(router chi.Router) {
		router.Get("/", logger.HanlderWithLogger(h.WithHashing(server.GzipMiddleware(h.DBPingHandler))))
	})
	router.Route("/updates", func(router chi.Router) {
		router.Post("/", logger.HanlderWithLogger(h.WithHashing(server.GzipMiddleware(h.MultipleUpdateHandler))))
	})
	router.Route("/update", func(router chi.Router) {
		router.Post("/", logger.HanlderWithLogger(h.WithHashing(server.GzipMiddleware(h.JSONUpdateHandler))))
		router.Route("/{mType}", func(router chi.Router) {
			router.Use(h.CheckMetricType)
			router.Route("/{mName}", func(router chi.Router) {
				router.Use(h.CheckMetricName)
				router.Post("/", logger.HanlderWithLogger(h.WithHashing(server.GzipMiddleware(func(rw http.ResponseWriter, r *http.Request) {
					logger.Log.Debug("no metric value has given")
					http.Error(rw, "incorrect metric value", http.StatusBadRequest)
				}))))
				router.Route("/{mValue}", func(router chi.Router) {
					router.Use(h.CheckMetricValue)
					router.Post("/", logger.HanlderWithLogger(h.WithHashing(server.GzipMiddleware(h.UpdateHandler))))
				})
			})
		})
	})
	router.Route("/value", func(router chi.Router) {
		router.Post("/", logger.HanlderWithLogger(h.WithHashing(server.GzipMiddleware(h.JSONGetHandler))))
		// router.Post("/", -> JSON VALUE GET HANDLER)
		router.Route("/{mType}", func(router chi.Router) {
			router.Use(h.CheckMetricType)
			router.Route("/{mName}", func(router chi.Router) {
				router.Use(h.CheckMetricName)
				router.Get("/", logger.HanlderWithLogger(h.WithHashing(server.GzipMiddleware(h.ValueHandler))))
			})
		})
	})
	return router
}
