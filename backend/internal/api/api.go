// Package api exposes the marketplace as a JSON REST API.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/krm-demo/timoha-projects/backend/internal/auth"
	"github.com/krm-demo/timoha-projects/backend/internal/catalog"
	"github.com/krm-demo/timoha-projects/backend/internal/home"
	"github.com/krm-demo/timoha-projects/backend/internal/payments"
	"github.com/krm-demo/timoha-projects/backend/internal/store"
)

type Server struct {
	Store  store.Store
	Tokens auth.Tokens
	Home   *home.Registry
	Now    func() time.Time
	Log    *slog.Logger
}

func New(s store.Store, tokens auth.Tokens, log *slog.Logger) *Server {
	return &Server{Store: s, Tokens: tokens, Home: home.NewRegistry(s), Now: time.Now, Log: log}
}

// Routes returns the API handler (everything under /api/).
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)

	mux.HandleFunc("GET /api/home", s.homeLayout)
	mux.HandleFunc("GET /api/home/{block}", s.homeBlock)

	mux.HandleFunc("GET /api/products", s.listProducts)
	mux.HandleFunc("GET /api/products/{slug}", s.getProduct)
	mux.HandleFunc("GET /api/search", s.search)

	mux.HandleFunc("POST /api/auth/register", s.register)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("GET /api/auth/me", s.requireUser(s.me))

	mux.HandleFunc("POST /api/checkout", s.requireUser(s.checkout))
	mux.HandleFunc("GET /api/orders", s.requireUser(s.orders))
	mux.HandleFunc("GET /api/orders/{id}", s.requireUser(s.order))

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeErr(w, http.StatusNotFound, "no such endpoint")
	})
	return mux
}

// ---------- helpers ----------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) internal(w http.ResponseWriter, r *http.Request, err error) {
	s.Log.Error("request failed", "path", r.URL.Path, "err", err)
	writeErr(w, http.StatusInternalServerError, "internal error")
}

func decode(r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

type ctxKey struct{}

func (s *Server) requireUser(next func(http.ResponseWriter, *http.Request, auth.Claims)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok {
			writeErr(w, http.StatusUnauthorized, "sign in required")
			return
		}
		c, err := s.Tokens.Verify(tok)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, err.Error())
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, c)), c)
	}
}

// ---------- handlers ----------

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "time": s.Now().UTC()})
}

func (s *Server) homeLayout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"blocks": s.Home.Layout()})
}

func (s *Server) homeBlock(w http.ResponseWriter, r *http.Request) {
	b, ok := s.Home.Get(r.PathValue("block"))
	if !ok {
		writeErr(w, http.StatusNotFound, "no such block")
		return
	}
	data, err := b.Load(r.Context())
	if err != nil {
		s.internal(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=30")
	writeJSON(w, http.StatusOK, data)
}

func parseQuery(r *http.Request) (catalog.Query, error) {
	v := r.URL.Query()
	q := catalog.Query{
		Q: strings.TrimSpace(v.Get("q")), Sort: v.Get("sort"),
		Platforms: v["platform"], Regions: v["region"], Types: v["type"], Genres: v["genre"],
	}
	for _, f := range []struct {
		key string
		dst **catalog.Money
	}{{"min", &q.Min}, {"max", &q.Max}} {
		if s := v.Get(f.key); s != "" {
			x, err := strconv.ParseFloat(s, 64)
			if err != nil || x < 0 {
				return q, errors.New(f.key + " must be a non-negative number")
			}
			m := catalog.Money(x * 100)
			*f.dst = &m
		}
	}
	q.Page, _ = strconv.Atoi(v.Get("page"))
	q.Limit, _ = strconv.Atoi(v.Get("limit"))
	return q, nil
}

func (s *Server) listProducts(w http.ResponseWriter, r *http.Request) {
	q, err := parseQuery(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	ps, err := s.Store.Products(r.Context())
	if err != nil {
		s.internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, catalog.List(ps, q))
}

func (s *Server) getProduct(w http.ResponseWriter, r *http.Request) {
	p, err := s.Store.Product(r.Context(), r.PathValue("slug"))
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "product not found")
		return
	}
	if err != nil {
		s.internal(w, r, err)
		return
	}
	ps, err := s.Store.Products(r.Context())
	if err != nil {
		s.internal(w, r, err)
		return
	}
	var related []catalog.Product
	for _, x := range ps {
		if x.ID != p.ID && (x.Genre == p.Genre || x.Platform == p.Platform) && len(related) < 5 {
			related = append(related, x)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"product": p, "related": related})
}

type suggestion struct {
	ID       int           `json:"id"`
	Slug     string        `json:"slug"`
	Name     string        `json:"name"`
	Platform string        `json:"platform"`
	Price    catalog.Money `json:"price"`
	Colors   []string      `json:"colors"`
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 20 {
		limit = 6
	}
	out := []suggestion{}
	if len(q) >= 2 {
		ps, err := s.Store.Products(r.Context())
		if err != nil {
			s.internal(w, r, err)
			return
		}
		hits := catalog.Search(ps, q)
		for _, p := range hits[:min(limit, len(hits))] {
			out = append(out, suggestion{p.ID, p.Slug, p.Name, p.Platform, p.FromPrice(), p.Colors})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"query": q, "results": out})
}

type userOut struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (s *Server) session(w http.ResponseWriter, status int, u store.User) {
	tok, err := s.Tokens.Issue(u.ID, u.Email, u.Name)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "could not issue token")
		return
	}
	writeJSON(w, status, map[string]any{"token": tok, "user": userOut{u.ID, u.Email, u.Name}})
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var in struct{ Name, Email, Password string }
	if err := decode(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	in.Name, in.Email = strings.TrimSpace(in.Name), strings.TrimSpace(in.Email)
	if _, err := mail.ParseAddress(in.Email); err != nil || strings.ContainsAny(in.Email, "<> ") {
		writeErr(w, http.StatusBadRequest, "enter a valid email")
		return
	}
	if in.Name == "" || len(in.Name) > 80 {
		writeErr(w, http.StatusBadRequest, "name is required (max 80 characters)")
		return
	}
	if len(in.Password) < 6 || len(in.Password) > 200 {
		writeErr(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		s.internal(w, r, err)
		return
	}
	u, err := s.Store.CreateUser(r.Context(), in.Email, in.Name, hash)
	if errors.Is(err, store.ErrConflict) {
		writeErr(w, http.StatusConflict, "an account with this email already exists")
		return
	}
	if err != nil {
		s.internal(w, r, err)
		return
	}
	s.session(w, http.StatusCreated, u)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Password string }
	if err := decode(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	u, err := s.Store.UserByEmail(r.Context(), in.Email)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		s.internal(w, r, err)
		return
	}
	if err != nil || !auth.CheckPassword(in.Password, u.PasswordHash) {
		writeErr(w, http.StatusUnauthorized, "wrong email or password")
		return
	}
	s.session(w, http.StatusOK, u)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request, c auth.Claims) {
	writeJSON(w, http.StatusOK, map[string]any{"user": userOut{c.UserID, c.Email, c.Name}})
}

func (s *Server) checkout(w http.ResponseWriter, r *http.Request, c auth.Claims) {
	var in struct {
		Items []store.LineRequest `json:"items"`
		Card  payments.Card       `json:"card"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	order, err := s.Store.PlaceOrder(r.Context(), c.UserID, in.Items, func(total catalog.Money) (string, error) {
		return payments.Charge(in.Card, total, s.Now())
	})
	var ve payments.ValidationError
	switch {
	case err == nil:
		s.Log.Info("order placed", "order", order.ID, "user", c.UserID, "total", float64(order.Total)/100, "card", payments.Describe(in.Card))
		writeJSON(w, http.StatusCreated, order)
	case errors.As(err, &ve), errors.Is(err, store.ErrBadLine):
		writeErr(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, payments.ErrDeclined), errors.Is(err, payments.ErrInsufficientFunds):
		writeErr(w, http.StatusPaymentRequired, err.Error())
	case errors.Is(err, store.ErrOutOfStock):
		writeErr(w, http.StatusConflict, err.Error())
	default:
		s.internal(w, r, err)
	}
}

func (s *Server) orders(w http.ResponseWriter, r *http.Request, c auth.Claims) {
	os, err := s.Store.Orders(r.Context(), c.UserID)
	if err != nil {
		s.internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"orders": os})
}

func (s *Server) order(w http.ResponseWriter, r *http.Request, c auth.Claims) {
	os, err := s.Store.Orders(r.Context(), c.UserID)
	if err != nil {
		s.internal(w, r, err)
		return
	}
	for _, o := range os {
		if o.ID == r.PathValue("id") {
			writeJSON(w, http.StatusOK, o)
			return
		}
	}
	writeErr(w, http.StatusNotFound, "order not found")
}
