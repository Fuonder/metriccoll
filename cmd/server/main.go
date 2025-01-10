package main

import (
	"github.com/Fuonder/metriccoll.git/internal/server"
	"github.com/Fuonder/metriccoll.git/internal/storage"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
)

func main() {
	err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Starting metric collector")
	if err = run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ms, err := storage.NewMemStorage()
	if err != nil {
		return err
	}
	handler := server.NewHandler(ms)

	log.Printf("Listening at %s\n", netAddr.String())
	return http.ListenAndServe(netAddr.String(), metricRouter(handler))
}

func metricRouter(h *server.Handler) chi.Router {
	log.Println("Entering router")
	router := chi.NewRouter()

	router.Use(h.CheckMethod)
	router.Use(h.CheckContentType)
	router.Get("/", h.RootHandler)
	router.Route("/update", func(router chi.Router) {
		router.Route("/{mType}", func(router chi.Router) {
			router.Use(h.CheckMetricType)
			router.Route("/{mName}", func(router chi.Router) {
				router.Use(h.CheckMetricName)
				router.Post("/", func(rw http.ResponseWriter, r *http.Request) {
					log.Println("no metric value has given")
					http.Error(rw, "incorrect metric value", http.StatusBadRequest)
				})
				router.Route("/{mValue}", func(router chi.Router) {
					router.Use(h.CheckMetricValue)
					router.Post("/", h.UpdateHandler)
				})
			})
		})
	})
	router.Route("/value", func(router chi.Router) {
		router.Route("/{mType}", func(router chi.Router) {
			router.Use(h.CheckMetricType)
			router.Route("/{mName}", func(router chi.Router) {
				router.Use(h.CheckMetricName)
				router.Get("/", h.ValueHandler)
			})
		})
	})
	return router
}
