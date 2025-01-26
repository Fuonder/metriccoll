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
)

func main() {
	err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}
	logger.Log.Info("Starting metric collector")
	if err = run(); err != nil {
		logger.Log.Fatal("", zap.Error(err))
	}
}

func run() error {
	if err := logger.Initialize(flagLogLevel); err != nil {
		return fmt.Errorf("method run: %v", err)
	}

	ms, err := storage.NewJSONStorage()
	if err != nil {
		return err
	}
	handler := server.NewHandler(ms)

	logger.Log.Info("Listening at",
		zap.String("Addr", netAddr.String()))
	return http.ListenAndServe(netAddr.String(), metricRouter(handler))
}

func metricRouter(h *server.Handler) chi.Router {
	logger.Log.Debug("Entering router")
	router := chi.NewRouter()

	router.Use(h.CheckMethod)
	router.Use(h.CheckContentType)
	router.Get("/", logger.HanlderWithLogger(h.RootHandler))
	router.Route("/update", func(router chi.Router) {
		//router.Post("/", -> HANDLER JSON UPDATE)
		router.Route("/{mType}", func(router chi.Router) {
			router.Use(h.CheckMetricType)
			router.Route("/{mName}", func(router chi.Router) {
				router.Use(h.CheckMetricName)
				router.Post("/", logger.HanlderWithLogger(func(rw http.ResponseWriter, r *http.Request) {
					logger.Log.Debug("no metric value has given")
					http.Error(rw, "incorrect metric value", http.StatusBadRequest)
				}))
				router.Route("/{mValue}", func(router chi.Router) {
					router.Use(h.CheckMetricValue)
					router.Post("/", logger.HanlderWithLogger(h.UpdateHandler))
				})
			})
		})
	})
	router.Route("/value", func(router chi.Router) {
		// router.Post("/", -> JSON VALUE GET HANDLER)
		router.Route("/{mType}", func(router chi.Router) {
			router.Use(h.CheckMetricType)
			router.Route("/{mName}", func(router chi.Router) {
				router.Use(h.CheckMetricName)
				router.Get("/", logger.HanlderWithLogger(h.ValueHandler))
			})
		})
	})
	return router
}
