package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/beadlima/research-api/monolith/internal/db"
	"github.com/beadlima/research-api/monolith/internal/handlers"
	"github.com/beadlima/research-api/monolith/internal/middleware"
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
	r.Use(middleware.PrometheusMetrics)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"monolith"}`))
	})
	r.Handle("/metrics", promhttp.Handler())

	r.Route("/users", func(r chi.Router) {
		r.Post("/", handlers.CreateUser(pool))
		r.Get("/", handlers.ListUsers(pool))
		r.Get("/{id}", handlers.GetUser(pool))
	})

	r.Route("/products", func(r chi.Router) {
		r.Post("/", handlers.CreateProduct(pool))
		r.Get("/", handlers.ListProducts(pool))
		r.Get("/{id}", handlers.GetProduct(pool))
	})

	r.Route("/orders", func(r chi.Router) {
		r.Post("/", handlers.CreateOrder(pool))
		r.Get("/", handlers.ListOrders(pool))
		r.Get("/{id}", handlers.GetOrder(pool))
	})

	r.Route("/inventory", func(r chi.Router) {
		r.Get("/{product_id}", handlers.GetInventory(pool))
		r.Patch("/{product_id}", handlers.UpdateInventory(pool))
	})

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Println("monolith listening on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
