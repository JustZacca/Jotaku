package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nzaccagnino/go-notes/internal/auth"
	"github.com/nzaccagnino/go-notes/internal/db"
)

type Server struct {
	db     *db.ServerDB
	jwt    *auth.JWTManager
	router *chi.Mux
}

type contextKey string

const userContextKey contextKey = "user"

func New(database *db.ServerDB, jwtManager *auth.JWTManager) *Server {
	s := &Server{
		db:     database,
		jwt:    jwtManager,
		router: chi.NewRouter(),
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Timeout(30 * time.Second))

	// Health check
	s.router.Get("/health", s.healthHandler)

	// Auth routes (public)
	s.router.Route("/api/auth", func(r chi.Router) {
		r.Post("/login", s.loginHandler)
		r.Post("/register", s.registerHandler)
	})

	// Protected routes
	s.router.Route("/api/notes", func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Get("/", s.listNotesHandler)
		r.Get("/{id}", s.getNoteHandler)
		r.Post("/", s.upsertNoteHandler)
		r.Delete("/{id}", s.deleteNoteHandler)
		r.Get("/sync", s.syncNotesHandler)
	})
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			jsonError(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			jsonError(w, "invalid authorization header", http.StatusUnauthorized)
			return
		}

		claims, err := s.jwt.Validate(parts[1])
		if err != nil {
			jsonError(w, "invalid token", http.StatusUnauthorized)
			return
		}

		user, err := s.db.GetUserByID(claims.UserID)
		if err != nil || user == nil || !user.Active {
			jsonError(w, "user not found or inactive", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getUserFromContext(r *http.Request) *db.User {
	user, _ := r.Context().Value(userContextKey).(*db.User)
	return user
}

func jsonResponse(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, message string, status int) {
	jsonResponse(w, map[string]string{"error": message}, status)
}
