package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/beadlima/research-api/monolith/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateUser(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
			return
		}
		if req.Name == "" || req.Email == "" {
			http.Error(w, `{"error":"name and email are required"}`, http.StatusBadRequest)
			return
		}

		var u models.User
		err := pool.QueryRow(r.Context(),
			`INSERT INTO users (name, email) VALUES ($1, $2)
			 RETURNING id, name, email, created_at`,
			req.Name, req.Email,
		).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)
		if err != nil {
			http.Error(w, `{"error":"could not create user"}`, http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, u)
	}
}

func GetUser(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
			return
		}

		var u models.User
		err = pool.QueryRow(r.Context(),
			`SELECT id, name, email, created_at FROM users WHERE id = $1`, id,
		).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)
		if err != nil {
			http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
			return
		}

		writeJSON(w, http.StatusOK, u)
	}
}

func ListUsers(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := pool.Query(r.Context(),
			`SELECT id, name, email, created_at FROM users ORDER BY id LIMIT 100`)
		if err != nil {
			http.Error(w, `{"error":"could not list users"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		users := make([]models.User, 0)
		for rows.Next() {
			var u models.User
			if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt); err != nil {
				http.Error(w, `{"error":"scan error"}`, http.StatusInternalServerError)
				return
			}
			users = append(users, u)
		}

		writeJSON(w, http.StatusOK, users)
	}
}
