package store

import (
	"context"
	"errors"
	"testing"

	"github.com/krm-demo/timoha-projects/backend/internal/catalog"
)

// runContract checks behaviour every Store implementation must share.
func runContract(t *testing.T, s Store) {
	ctx := context.Background()

	ps, err := s.Products(ctx)
	if err != nil || len(ps) < 10 {
		t.Fatalf("products: %v (n=%d)", err, len(ps))
	}
	p, err := s.Product(ctx, ps[0].Slug)
	if err != nil || p.ID != ps[0].ID || len(p.Offers) == 0 {
		t.Fatalf("product by slug: %v", err)
	}
	if _, err := s.Product(ctx, "no-such-thing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing product: %v", err)
	}

	u, err := s.CreateUser(ctx, "Tim@Example.com", "Tim", "hash")
	if err != nil || u.ID == 0 || u.Email != "tim@example.com" {
		t.Fatalf("create user: %v %+v", err, u)
	}
	if _, err := s.CreateUser(ctx, "tim@example.com", "Tim2", "hash"); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate email: %v", err)
	}
	if got, err := s.UserByEmail(ctx, "TIM@example.com"); err != nil || got.ID != u.ID || got.PasswordHash != "hash" {
		t.Fatalf("user by email: %v %+v", err, got)
	}

	offer := p.Offers[0]
	lines := []LineRequest{{p.ID, offer.Seller, 1}, {p.ID, offer.Seller, 1}} // merged into qty 2

	// a failed payment must not change anything
	if _, err := s.PlaceOrder(ctx, u.ID, lines, func(catalog.Money) (string, error) { return "", errors.New("declined") }); err == nil {
		t.Fatal("expected payment error")
	}
	if after, _ := s.Product(ctx, p.Slug); after.Sold != p.Sold {
		t.Fatal("declined payment changed the product")
	}

	var charged catalog.Money
	o, err := s.PlaceOrder(ctx, u.ID, lines, func(m catalog.Money) (string, error) { charged = m; return "pay_1", nil })
	if err != nil {
		t.Fatalf("place order: %v", err)
	}
	wantSub := offer.Price * 2
	if o.Subtotal != wantSub || o.Total != wantSub+ServiceFee(wantSub) || charged != o.Total {
		t.Fatalf("totals: %+v charged=%v", o, charged)
	}
	if len(o.Items) != 1 || len(o.Items[0].Keys) != 2 || o.Items[0].Keys[0] == o.Items[0].Keys[1] {
		t.Fatalf("keys: %+v", o.Items)
	}
	after, _ := s.Product(ctx, p.Slug)
	if after.Sold != p.Sold+2 {
		t.Fatalf("sold not updated: %d -> %d", p.Sold, after.Sold)
	}
	if ao, _, _ := after.Offer(offer.Seller); ao.Stock != offer.Stock-2 {
		t.Fatalf("stock not decremented: %d -> %d", offer.Stock, ao.Stock)
	}

	// out of stock and bad lines
	if _, err := s.PlaceOrder(ctx, u.ID, []LineRequest{{p.ID, offer.Seller, 10}, {p.ID, offer.Seller, 10}}, okCharge); !errors.Is(err, ErrBadLine) && !errors.Is(err, ErrOutOfStock) {
		t.Fatalf("too many: %v", err)
	}
	if _, err := s.PlaceOrder(ctx, u.ID, []LineRequest{{p.ID, "NoSuchSeller", 1}}, okCharge); !errors.Is(err, ErrBadLine) {
		t.Fatalf("bad seller: %v", err)
	}

	orders, err := s.Orders(ctx, u.ID)
	if err != nil || len(orders) != 1 || orders[0].ID != o.ID || len(orders[0].Items[0].Keys) != 2 {
		t.Fatalf("orders: %v %+v", err, orders)
	}
	if other, _ := s.Orders(ctx, u.ID+999); len(other) != 0 {
		t.Fatal("orders leaked to another user")
	}
}

func okCharge(catalog.Money) (string, error) { return "pay_ok", nil }
