// Package store is the persistence layer. Store is implemented by an
// embedded JSON-file database (default, zero dependencies) and by PostgreSQL
// (build tag "postgres").
package store

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/krm-demo/timoha-projects/backend/internal/catalog"
)

//go:embed seed/products.json
var seedProducts []byte

// SeedProducts returns the demo catalog shipped with the binary.
func SeedProducts() ([]catalog.Product, error) {
	var ps []catalog.Product
	if err := json.Unmarshal(seedProducts, &ps); err != nil {
		return nil, fmt.Errorf("seed: %w", err)
	}
	for i := range ps {
		ps[i].SortOffers()
	}
	return ps, nil
}

var (
	ErrNotFound   = errors.New("not found")
	ErrConflict   = errors.New("already exists")
	ErrOutOfStock = errors.New("not enough stock")
	ErrBadLine    = errors.New("invalid order line")
)

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"passwordHash,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

// LineRequest is one cart line sent to checkout. Price is never taken from
// the client; it is read from the offer inside the transaction.
type LineRequest struct {
	ProductID int    `json:"productId"`
	Seller    string `json:"seller"`
	Qty       int    `json:"qty"`
}

type OrderItem struct {
	ProductID int           `json:"productId"`
	Slug      string        `json:"slug"`
	Name      string        `json:"name"`
	Platform  string        `json:"platform"`
	Region    string        `json:"region"`
	Seller    string        `json:"seller"`
	Price     catalog.Money `json:"price"`
	Qty       int           `json:"qty"`
	Keys      []string      `json:"keys"`
}

type Order struct {
	ID        string        `json:"id"`
	UserID    int64         `json:"userId"`
	CreatedAt time.Time     `json:"date"`
	Subtotal  catalog.Money `json:"subtotal"`
	Fee       catalog.Money `json:"fee"`
	Total     catalog.Money `json:"total"`
	PaymentID string        `json:"paymentId"`
	Items     []OrderItem   `json:"items"`
}

// ChargeFunc takes the payment for the computed total. If it fails, the
// order is not created and stock is untouched.
type ChargeFunc func(total catalog.Money) (paymentID string, err error)

type Store interface {
	Products(ctx context.Context) ([]catalog.Product, error)
	Product(ctx context.Context, slug string) (catalog.Product, error)
	CreateUser(ctx context.Context, email, name, passwordHash string) (User, error)
	UserByEmail(ctx context.Context, email string) (User, error)
	PlaceOrder(ctx context.Context, userID int64, lines []LineRequest, charge ChargeFunc) (Order, error)
	Orders(ctx context.Context, userID int64) ([]Order, error)
	Close() error
}

// ServiceFee is the marketplace fee: 5% of the subtotal, rounded to the cent.
func ServiceFee(subtotal catalog.Money) catalog.Money {
	// Integer arithmetic: subtotal*5/100 with +50 so it rounds half up
	// instead of truncating (1999c → 99.95c → 100c, not 99c).
	return (subtotal*5 + 50) / 100
}

// ValidateLines merges duplicates and rejects empty or silly carts.
func ValidateLines(lines []LineRequest) ([]LineRequest, error) {
	if len(lines) == 0 {
		return nil, fmt.Errorf("%w: cart is empty", ErrBadLine)
	}
	idx := map[string]int{}
	var out []LineRequest
	for _, l := range lines {
		if l.Qty <= 0 || l.Qty > 10 {
			return nil, fmt.Errorf("%w: quantity must be 1-10", ErrBadLine)
		}
		k := fmt.Sprint(l.ProductID, "|", l.Seller)
		if i, ok := idx[k]; ok {
			out[i].Qty += l.Qty
			continue
		}
		idx[k] = len(out)
		out = append(out, l)
	}
	for _, l := range out {
		if l.Qty > 10 {
			return nil, fmt.Errorf("%w: at most 10 keys per offer", ErrBadLine)
		}
	}
	// Always lock offers in the same order. If buyer A locks offer 1 then 2
	// while buyer B locks 2 then 1, each waits for the other forever (a
	// deadlock; Postgres would kill one of the transactions). Sorting by
	// (product, seller) makes every checkout take locks in one global order.
	sort.Slice(out, func(i, j int) bool {
		if out[i].ProductID != out[j].ProductID {
			return out[i].ProductID < out[j].ProductID
		}
		return out[i].Seller < out[j].Seller
	})
	return out, nil
}

const keyAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no 0/O/1/I

// NewKey generates a product key like "7KQ2M-XW9PA-3HTRN".
func NewKey() string {
	b := make([]byte, 15)
	_, _ = rand.Read(b)
	var s strings.Builder
	for i, c := range b {
		if i > 0 && i%5 == 0 {
			s.WriteByte('-')
		}
		// The alphabet has 32 symbols and 256 is divisible by 32, so
		// byte % 32 is perfectly uniform (no "modulo bias").
		s.WriteByte(keyAlphabet[int(c)%len(keyAlphabet)])
	}
	return s.String()
}

// NewOrderID returns an id like "TM-20261124-8F3K2Q".
func NewOrderID(now time.Time) string {
	return "TM-" + now.UTC().Format("20060102") + "-" + NewKey()[:5]
}

// OpenPostgres is set by pgstore.go when built with -tags postgres.
var OpenPostgres func(ctx context.Context, url string) (Store, error)
