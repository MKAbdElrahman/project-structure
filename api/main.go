package main

import (
	"context"
	"counter/db"
	"counter/handler"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {

	// Establish connection pool to PostgreSQL.
	pool, err := pgxpool.New(context.Background(), os.Getenv("DB_DSN"))
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	sessionStore := db.NewPostgresSessionStore(pool, 30*time.Minute)
	// Initialize a new session manager and configure the session lifetime.

	sessionManager := scs.New()
	sessionManager.Store = sessionStore

	sessionManager.Lifetime = 24 * time.Hour

	errLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelError,
	}))

	sessionManager.Lifetime = 24 * time.Hour

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	homeHandler := handler.NewHomeHandler(errLogger, sessionManager)

	r.Get("/", homeHandler.HandleGet)
	r.Post("/", homeHandler.HandlePost)

	http.ListenAndServe(":3000", sessionManager.LoadAndSave(r))
}
