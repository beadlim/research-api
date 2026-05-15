package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/beadlima/research-api/stages/05-full-microservices/orders-service/internal/clients"
	"github.com/beadlima/research-api/stages/05-full-microservices/orders-service/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateOrder(pool *pgxpool.Pool, users *clients.UsersClient, products *clients.ProductsClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.CreateOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
			return
		}
		if req.UserID == 0 || len(req.Items) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id and items are required"})
			return
		}

		// HTTP call → users-service (overhead inter-serviços mensurável aqui)
		exists, err := users.Exists(r.Context(), req.UserID)
		if err != nil || !exists {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
			return
		}

		tx, err := pool.Begin(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not start transaction"})
			return
		}
		defer tx.Rollback(r.Context())

		var orderID int
		err = tx.QueryRow(r.Context(),
			`INSERT INTO orders (user_id, status) VALUES ($1, 'pending') RETURNING id`,
			req.UserID,
		).Scan(&orderID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not create order"})
			return
		}

		var total float64
		items := make([]models.OrderItem, 0, len(req.Items))

		for _, reqItem := range req.Items {
			// HTTP call → products-service (overhead mensurável aqui)
			price, err := products.GetPrice(r.Context(), reqItem.ProductID)
			if err != nil {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
				return
			}

			var item models.OrderItem
			err = tx.QueryRow(r.Context(),
				`INSERT INTO order_items (order_id, product_id, quantity, price)
				 VALUES ($1, $2, $3, $4)
				 RETURNING id, order_id, product_id, quantity, price`,
				orderID, reqItem.ProductID, reqItem.Quantity, price,
			).Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.Price)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not insert order item"})
				return
			}

			total += price * float64(reqItem.Quantity)
			items = append(items, item)
		}

		var order models.Order
		err = tx.QueryRow(r.Context(),
			`UPDATE orders SET total = $1 WHERE id = $2
			 RETURNING id, user_id, total, status, created_at`,
			total, orderID,
		).Scan(&order.ID, &order.UserID, &order.Total, &order.Status, &order.CreatedAt)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not update order total"})
			return
		}

		if err := tx.Commit(r.Context()); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not commit"})
			return
		}

		order.Items = items
		writeJSON(w, http.StatusCreated, order)
	}
}

func GetOrder(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
			return
		}

		var o models.Order
		err = pool.QueryRow(r.Context(),
			`SELECT id, user_id, total, status, created_at FROM orders WHERE id = $1`, id,
		).Scan(&o.ID, &o.UserID, &o.Total, &o.Status, &o.CreatedAt)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "order not found"})
			return
		}

		rows, err := pool.Query(r.Context(),
			`SELECT id, order_id, product_id, quantity, price FROM order_items WHERE order_id = $1`, id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not fetch items"})
			return
		}
		defer rows.Close()

		o.Items = make([]models.OrderItem, 0)
		for rows.Next() {
			var item models.OrderItem
			if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.Price); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan error"})
				return
			}
			o.Items = append(o.Items, item)
		}

		writeJSON(w, http.StatusOK, o)
	}
}

func ListOrders(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := pool.Query(r.Context(),
			`SELECT id, user_id, total, status, created_at FROM orders ORDER BY id DESC LIMIT 100`)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not list orders"})
			return
		}
		defer rows.Close()

		orders := make([]models.Order, 0)
		for rows.Next() {
			var o models.Order
			if err := rows.Scan(&o.ID, &o.UserID, &o.Total, &o.Status, &o.CreatedAt); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan error"})
				return
			}
			orders = append(orders, o)
		}

		writeJSON(w, http.StatusOK, orders)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
