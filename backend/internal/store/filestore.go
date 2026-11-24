package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/krm-demo/timoha-projects/backend/internal/catalog"
)

// FileStore keeps everything in memory and writes a JSON snapshot to disk
// after every change (atomic rename). One mutex gives serialisable
// transactions, which is plenty for a single-instance demo.
type FileStore struct {
	mu   sync.RWMutex
	path string // "" = memory only
	db   fileDB
	now  func() time.Time
}

type fileDB struct {
	Products   []catalog.Product `json:"products"`
	Users      []User            `json:"users"`
	Orders     []Order           `json:"orders"`
	NextUserID int64             `json:"nextUserId"`
}

// OpenFile loads path, or seeds a new database there if it doesn't exist.
func OpenFile(path string) (*FileStore, error) {
	s := &FileStore{path: path, now: time.Now}
	if path != "" {
		raw, err := os.ReadFile(path)
		switch {
		case err == nil:
			if err := json.Unmarshal(raw, &s.db); err != nil {
				return nil, fmt.Errorf("read %s: %w", path, err)
			}
			return s, nil
		case !errors.Is(err, os.ErrNotExist):
			return nil, err
		}
	}
	ps, err := SeedProducts()
	if err != nil {
		return nil, err
	}
	s.db = fileDB{Products: ps, NextUserID: 1}
	return s, s.save()
}

func (s *FileStore) save() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	raw, err := json.Marshal(s.db)
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *FileStore) Close() error { return nil }

func cloneProduct(p catalog.Product) catalog.Product {
	p.Offers = slices.Clone(p.Offers)
	p.Colors = slices.Clone(p.Colors)
	return p
}

func (s *FileStore) Products(context.Context) ([]catalog.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]catalog.Product, len(s.db.Products))
	for i, p := range s.db.Products {
		out[i] = cloneProduct(p)
	}
	return out, nil
}

func (s *FileStore) Product(_ context.Context, slug string) (catalog.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.db.Products {
		if p.Slug == slug {
			return cloneProduct(p), nil
		}
	}
	return catalog.Product{}, ErrNotFound
}

func (s *FileStore) CreateUser(_ context.Context, email, name, hash string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	email = strings.ToLower(strings.TrimSpace(email))
	for _, u := range s.db.Users {
		if u.Email == email {
			return User{}, ErrConflict
		}
	}
	u := User{ID: s.db.NextUserID, Email: email, Name: name, PasswordHash: hash, CreatedAt: s.now().UTC()}
	s.db.NextUserID++
	s.db.Users = append(s.db.Users, u)
	return u, s.save()
}

func (s *FileStore) UserByEmail(_ context.Context, email string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	email = strings.ToLower(strings.TrimSpace(email))
	for _, u := range s.db.Users {
		if u.Email == email {
			return u, nil
		}
	}
	return User{}, ErrNotFound
}

func (s *FileStore) PlaceOrder(_ context.Context, userID int64, req []LineRequest, charge ChargeFunc) (Order, error) {
	lines, err := ValidateLines(req)
	if err != nil {
		return Order{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	type pick struct{ pi, oi int }
	picks := make([]pick, len(lines))
	order := Order{UserID: userID, CreatedAt: s.now().UTC()}
	for i, l := range lines {
		pi := slices.IndexFunc(s.db.Products, func(p catalog.Product) bool { return p.ID == l.ProductID })
		if pi < 0 {
			return Order{}, fmt.Errorf("%w: unknown product %d", ErrBadLine, l.ProductID)
		}
		p := s.db.Products[pi]
		o, oi, ok := p.Offer(l.Seller)
		if !ok {
			return Order{}, fmt.Errorf("%w: %s is not sold by %s", ErrBadLine, p.Name, l.Seller)
		}
		if o.Stock < l.Qty {
			return Order{}, fmt.Errorf("%w: %s by %s has %d left", ErrOutOfStock, p.Name, l.Seller, o.Stock)
		}
		picks[i] = pick{pi, oi}
		order.Subtotal += o.Price * catalog.Money(l.Qty)
		order.Items = append(order.Items, OrderItem{ProductID: p.ID, Slug: p.Slug, Name: p.Name,
			Platform: p.Platform, Region: p.Region, Seller: o.Seller, Price: o.Price, Qty: l.Qty})
	}
	order.Fee = ServiceFee(order.Subtotal)
	order.Total = order.Subtotal + order.Fee

	payID, err := charge(order.Total)
	if err != nil {
		return Order{}, err // nothing was changed
	}
	order.PaymentID = payID
	order.ID = NewOrderID(order.CreatedAt)
	for i, l := range lines {
		p := &s.db.Products[picks[i].pi]
		p.Offers[picks[i].oi].Stock -= l.Qty
		p.Sold += l.Qty
		p.SortOffers()
		for range l.Qty {
			order.Items[i].Keys = append(order.Items[i].Keys, NewKey())
		}
	}
	s.db.Orders = append(s.db.Orders, order)
	return order, s.save()
}

func (s *FileStore) Orders(_ context.Context, userID int64) ([]Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Order{}
	for i := len(s.db.Orders) - 1; i >= 0; i-- { // newest first
		if s.db.Orders[i].UserID == userID {
			out = append(out, s.db.Orders[i])
		}
	}
	return out, nil
}
