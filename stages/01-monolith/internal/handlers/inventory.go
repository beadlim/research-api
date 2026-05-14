package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/beadlima/research-api/monolith/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func GetInventory(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		productID, err := strconv.Atoi(chi.URLParam(r, "product_id"))
		if err != nil {
			http.Error(w, `{"error":"invalid product_id"}`, http.StatusBadRequest)
			return
		}

		var inv models.Inventory
		err = pool.QueryRow(r.Context(),
			`SELECT id, product_id, quantity, updated_at FROM inventory WHERE product_id = $1`,
			productID,
		).Scan(&inv.ID, &inv.ProductID, &inv.Quantity, &inv.UpdatedAt)
		if err != nil {
			http.Error(w, `{"error":"inventory not found"}`, http.StatusNotFound)
			return
		}

		writeJSON(w, http.StatusOK, inv)
	}
}

func UpdateInventory(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		productID, err := strconv.Atoi(chi.URLParam(r, "product_id"))
		if err != nil {
			http.Error(w, `{"error":"invalid product_id"}`, http.StatusBadRequest)
			return
		}

		var req models.UpdateInventoryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
			return
		}

		var inv models.Inventory
		err = pool.QueryRow(r.Context(),
			`INSERT INTO inventory (product_id, quantity) VALUES ($1, $2)
			 ON CONFLICT (product_id) DO UPDATE
			 SET quantity = $2, updated_at = NOW()
			 RETURNING id, product_id, quantity, updated_at`,
			productID, req.Quantity,
		).Scan(&inv.ID, &inv.ProductID, &inv.Quantity, &inv.UpdatedAt)
		if err != nil {
			http.Error(w, `{"error":"could not update inventory"}`, http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, inv)
	}
}
