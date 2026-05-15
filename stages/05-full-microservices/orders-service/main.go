package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/beadlima/research-api/stages/05-full-microservices/orders-service/internal/clients"
	"github.com/beadlima/research-api/stages/05-full-microservices/orders-service/internal/db"
	"github.com/beadlima/research-api/stages/05-full-microservices/orders-service/internal/handlers"
	"github.com/beadlima/research-api/stages/05-full-microservices/orders-service/internal/middleware"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/research?sslmode=disable"
	}
	usersURL := os.Getenv("USERS_SERVICE_URL")
	if usersURL == "" {
		usersURL = "http://localhost:8081"
	}
	productsURL := os.Getenv("PRODUCTS_SERVICE_URL")
	if productsURL == "" {
		productsURL = "http://localhost:8082"
	}

	pool, err := db.Connect(dbURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(pool); err != nil {
		log.Fatalf("db migrate: %v", err)
	}

	usersClient := clients.NewUsersClient(usersURL)
	productsClient := clients.NewProductsClient(productsURL)

	r := chi.NewRouter()
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.PrometheusMetrics("orders-service"))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"degraded","service":"orders-service","db":"down"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"orders-service"}`))
	})
	r.Handle("/metrics", promhttp.Handler())

	r.Route("/orders", func(r chi.Router) {
		r.Post("/", handlers.CreateOrder(pool, usersClient, productsClient))
		r.Get("/", handlers.ListOrders(pool))
		r.Get("/{id}", handlers.GetOrder(pool))
	})

	srv := &http.Server{
		Addr:         ":8083",
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Println("orders-service listening on :8083")
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
