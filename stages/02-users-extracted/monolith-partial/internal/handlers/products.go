package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/beadlima/research-api/monolith-partial/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateProduct(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.CreateProductRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
			return
		}
		if req.Name == "" || req.Price <= 0 {
			http.Error(w, `{"error":"name and positive price are required"}`, http.StatusBadRequest)
			return
		}

		var p models.Product
		err := pool.QueryRow(r.Context(),
			`INSERT INTO products (name, price) VALUES ($1, $2)
			 RETURNING id, name, price, created_at`,
			req.Name, req.Price,
		).Scan(&p.ID, &p.Name, &p.Price, &p.CreatedAt)
		if err != nil {
			http.Error(w, `{"error":"could not create product"}`, http.StatusInternalServerError)
			return
		}

		// create inventory entry with 0 stock
		pool.Exec(r.Context(),
			`INSERT INTO inventory (product_id, quantity) VALUES ($1, 0) ON CONFLICT DO NOTHING`,
			p.ID,
		)

		writeJSON(w, http.StatusCreated, p)
	}
}

func GetProduct(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
			return
		}

		var p models.Product
		err = pool.QueryRow(r.Context(),
			`SELECT id, name, price, created_at FROM products WHERE id = $1`, id,
		).Scan(&p.ID, &p.Name, &p.Price, &p.CreatedAt)
		if err != nil {
			http.Error(w, `{"error":"product not found"}`, http.StatusNotFound)
			return
		}

		writeJSON(w, http.StatusOK, p)
	}
}

func ListProducts(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := pool.Query(r.Context(),
			`SELECT id, name, price, created_at FROM products ORDER BY id LIMIT 100`)
		if err != nil {
			http.Error(w, `{"error":"could not list products"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		products := make([]models.Product, 0)
		for rows.Next() {
			var p models.Product
			if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.CreatedAt); err != nil {
				http.Error(w, `{"error":"scan error"}`, http.StatusInternalServerError)
				return
			}
			products = append(products, p)
		}

		writeJSON(w, http.StatusOK, products)
	}
}
