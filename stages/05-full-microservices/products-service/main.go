package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/beadlima/research-api/stages/05-full-microservices/products-service/internal/db"
	"github.com/beadlima/research-api/stages/05-full-microservices/products-service/internal/handlers"
	"github.com/beadlima/research-api/stages/05-full-microservices/products-service/internal/middleware"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/research?sslmode=disable"
	}

	pool, err := db.Connect(dbURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(pool); err != nil {
		log.Fatalf("db migrate: %v", err)
	}

	r := chi.NewRouter()
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.PrometheusMetrics("products-service"))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"degraded","service":"products-service","db":"down"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"products-service"}`))
	})
	r.Handle("/metrics", promhttp.Handler())

	r.Route("/products", func(r chi.Router) {
		r.Post("/", handlers.CreateProduct(pool))
		r.Get("/", handlers.ListProducts(pool))
		r.Get("/{id}", handlers.GetProduct(pool))
	})

	srv := &http.Server{
		Addr:         ":8082",
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Println("products-service listening on :8082")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
