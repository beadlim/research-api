package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/beadlima/research-api/users-service/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateUser(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
			return
		}
		if req.Name == "" || req.Email == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and email are required"})
			return
		}

		var u models.User
		err := pool.QueryRow(r.Context(),
			`INSERT INTO users (name, email) VALUES ($1, $2)
			 RETURNING id, name, email, created_at`,
			req.Name, req.Email,
		).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not create user"})
			return
		}

		writeJSON(w, http.StatusCreated, u)
	}
}

func GetUser(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
			return
		}

		var u models.User
		err = pool.QueryRow(r.Context(),
			`SELECT id, name, email, created_at FROM users WHERE id = $1`, id,
		).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
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
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not list users"})
			return
		}
		defer rows.Close()

		users := make([]models.User, 0)
		for rows.Next() {
			var u models.User
			if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan error"})
				return
			}
			users = append(users, u)
		}

		writeJSON(w, http.StatusOK, users)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
