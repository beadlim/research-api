package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/beadlima/research-api/monolith-partial-03/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateOrder(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.CreateOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
			return
		}
		if req.UserID == 0 || len(req.Items) == 0 {
			http.Error(w, `{"error":"user_id and items are required"}`, http.StatusBadRequest)
			return
		}

		tx, err := pool.Begin(r.Context())
		if err != nil {
			http.Error(w, `{"error":"could not start transaction"}`, http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(r.Context())

		// verify user exists
		var userExists bool
		tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)`, req.UserID).Scan(&userExists)
		if !userExists {
			http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
			return
		}

		// insert order
		var orderID int
		err = tx.QueryRow(r.Context(),
			`INSERT INTO orders (user_id, status) VALUES ($1, 'pending') RETURNING id`,
			req.UserID,
		).Scan(&orderID)
		if err != nil {
			http.Error(w, `{"error":"could not create order"}`, http.StatusInternalServerError)
			return
		}

		var total float64
		items := make([]models.OrderItem, 0, len(req.Items))

		for _, reqItem := range req.Items {
			// fetch product price (direct DB call — no network hop, unlike microservices)
			var price float64
			err := tx.QueryRow(r.Context(),
				`SELECT price FROM products WHERE id = $1`, reqItem.ProductID,
			).Scan(&price)
			if err != nil {
				http.Error(w, `{"error":"product not found"}`, http.StatusNotFound)
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
				http.Error(w, `{"error":"could not insert order item"}`, http.StatusInternalServerError)
				return
			}

			total += price * float64(reqItem.Quantity)
			items = append(items, item)
		}

		// update order total
		var order models.Order
		err = tx.QueryRow(r.Context(),
			`UPDATE orders SET total = $1 WHERE id = $2
			 RETURNING id, user_id, total, status, created_at`,
			total, orderID,
		).Scan(&order.ID, &order.UserID, &order.Total, &order.Status, &order.CreatedAt)
		if err != nil {
			http.Error(w, `{"error":"could not update order total"}`, http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(r.Context()); err != nil {
			http.Error(w, `{"error":"could not commit transaction"}`, http.StatusInternalServerError)
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
			http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
			return
		}

		var o models.Order
		err = pool.QueryRow(r.Context(),
			`SELECT id, user_id, total, status, created_at FROM orders WHERE id = $1`, id,
		).Scan(&o.ID, &o.UserID, &o.Total, &o.Status, &o.CreatedAt)
		if err != nil {
			http.Error(w, `{"error":"order not found"}`, http.StatusNotFound)
			return
		}

		rows, err := pool.Query(r.Context(),
			`SELECT id, order_id, product_id, quantity, price FROM order_items WHERE order_id = $1`, id)
		if err != nil {
			http.Error(w, `{"error":"could not fetch items"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		o.Items = make([]models.OrderItem, 0)
		for rows.Next() {
			var item models.OrderItem
			if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.Price); err != nil {
				http.Error(w, `{"error":"scan error"}`, http.StatusInternalServerError)
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
			http.Error(w, `{"error":"could not list orders"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		orders := make([]models.Order, 0)
		for rows.Next() {
			var o models.Order
			if err := rows.Scan(&o.ID, &o.UserID, &o.Total, &o.Status, &o.CreatedAt); err != nil {
				http.Error(w, `{"error":"scan error"}`, http.StatusInternalServerError)
				return
			}
			orders = append(orders, o)
		}

		writeJSON(w, http.StatusOK, orders)
	}
}
