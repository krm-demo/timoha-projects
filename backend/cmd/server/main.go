// Command server runs the Timoha Market REST API and (optionally) serves the
// static frontend from the same origin.
//
// Configuration (environment):
//
//	PORT              listen port (default 8080)
//	DATABASE_URL      postgres://... ; needs a binary built with -tags postgres
//	DATA_FILE         JSON-file database path when DATABASE_URL is empty (default data/db.json)
//	JWT_SECRET        token signing secret (random per start if empty: sessions reset on restart)
//	ALLOWED_ORIGINS   comma-separated CORS origins (default *)
//	STATIC_DIR        frontend directory to serve at / (default ../web, "" to disable)
package main

import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/krm-demo/timoha-projects/backend/internal/api"
	"github.com/krm-demo/timoha-projects/backend/internal/auth"
	"github.com/krm-demo/timoha-projects/backend/internal/store"
)

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var st store.Store
	if url := os.Getenv("DATABASE_URL"); url != "" {
		if store.OpenPostgres == nil {
			return errors.New("DATABASE_URL is set but this binary was built without -tags postgres")
		}
		s, err := store.OpenPostgres(ctx, url)
		if err != nil {
			return err
		}
		st = s
		log.Info("database", "kind", "postgres")
	} else {
		path := env("DATA_FILE", "data/db.json")
		s, err := store.OpenFile(path)
		if err != nil {
			return err
		}
		st = s
		log.Info("database", "kind", "json-file", "path", path)
	}
	defer st.Close()

	secret := []byte(os.Getenv("JWT_SECRET"))
	if len(secret) == 0 {
		secret = make([]byte, 32)
		_, _ = rand.Read(secret)
		log.Warn("JWT_SECRET not set: using a random secret, sessions end on restart")
	}

	srv := api.New(st, auth.Tokens{Secret: secret}, log)
	mux := http.NewServeMux()
	mux.Handle("/api/", srv.Routes())
	if dir := env("STATIC_DIR", "../web"); dir != "" {
		if _, err := os.Stat(dir); err == nil {
			mux.Handle("/", http.FileServer(http.Dir(dir)))
			log.Info("serving frontend", "dir", dir)
		}
	}
	origins := strings.Split(env("ALLOWED_ORIGINS", "*"), ",")
	handler := api.Logging(log, api.CORS(origins, api.RateLimit(20, mux)))

	hs := &http.Server{
		Addr:              ":" + env("PORT", "8080"),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	errc := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", hs.Addr)
		errc <- hs.ListenAndServe()
	}()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	log.Info("shutting down")
	return hs.Shutdown(shutdown)
}
