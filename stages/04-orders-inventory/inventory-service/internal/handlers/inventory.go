package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/beadlima/research-api/inventory-service/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func GetInventory(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		productID, err := strconv.Atoi(chi.URLParam(r, "product_id"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product_id"})
			return
		}

		var inv models.Inventory
		err = pool.QueryRow(r.Context(),
			`SELECT id, product_id, quantity, updated_at FROM inventory WHERE product_id = $1`,
			productID,
		).Scan(&inv.ID, &inv.ProductID, &inv.Quantity, &inv.UpdatedAt)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "inventory not found"})
			return
		}

		writeJSON(w, http.StatusOK, inv)
	}
}

func UpdateInventory(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		productID, err := strconv.Atoi(chi.URLParam(r, "product_id"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product_id"})
			return
		}

		var req models.UpdateInventoryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
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
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not update inventory"})
			return
		}

		writeJSON(w, http.StatusOK, inv)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
