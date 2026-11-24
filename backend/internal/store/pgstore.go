//go:build postgres

package store

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver

	"github.com/krm-demo/timoha-projects/backend/internal/catalog"
)

//go:embed schema.sql
var schemaSQL string

func init() {
	OpenPostgres = func(ctx context.Context, url string) (Store, error) { return openPG(ctx, url) }
}

// PGStore is the PostgreSQL implementation. The catalog is small, so listing
// and fuzzy search run in Go over Products(); checkout is one transaction
// with row locks on the affected offers.
type PGStore struct {
	db  *sql.DB
	now func() time.Time
}

func openPG(ctx context.Context, url string) (*PGStore, error) {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	s := &PGStore{db: db, now: time.Now}
	if _, err := db.ExecContext(ctx, schemaSQL); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, s.seed(ctx)
}

func (s *PGStore) Close() error { return s.db.Close() }

func (s *PGStore) seed(ctx context.Context) error {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM products`).Scan(&n); err != nil || n > 0 {
		return err
	}
	ps, err := SeedProducts()
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, p := range ps {
		colors, _ := json.Marshal(p.Colors)
		if _, err := tx.ExecContext(ctx, `INSERT INTO products (id,slug,name,platform,region,type,genre,year,msrp_cents,colors,rating,sold,description)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
			p.ID, p.Slug, p.Name, p.Platform, p.Region, p.Type, p.Genre, p.Year, int64(p.MSRP), string(colors), p.Rating, p.Sold, p.Description); err != nil {
			return err
		}
		for _, o := range p.Offers {
			if _, err := tx.ExecContext(ctx, `INSERT INTO offers (product_id,seller,rating,reviews,price_cents,stock,delivery) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
				p.ID, o.Seller, o.Rating, o.Reviews, int64(o.Price), o.Stock, o.Delivery); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

const productCols = `id,slug,name,platform,region,type,genre,year,msrp_cents,colors,rating,sold,description`

func (s *PGStore) loadProducts(ctx context.Context, where string, args ...any) ([]catalog.Product, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+productCols+` FROM products `+where+` ORDER BY id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ps []catalog.Product
	idx := map[int]int{}
	for rows.Next() {
		var p catalog.Product
		var msrp int64
		var colors string
		if err := rows.Scan(&p.ID, &p.Slug, &p.Name, &p.Platform, &p.Region, &p.Type, &p.Genre, &p.Year, &msrp, &colors, &p.Rating, &p.Sold, &p.Description); err != nil {
			return nil, err
		}
		p.MSRP = catalog.Money(msrp)
		_ = json.Unmarshal([]byte(colors), &p.Colors)
		idx[p.ID] = len(ps)
		ps = append(ps, p)
	}
	if err := rows.Err(); err != nil || len(ps) == 0 {
		return ps, err
	}
	ids := make([]string, 0, len(ps))
	for _, p := range ps {
		ids = append(ids, fmt.Sprint(p.ID))
	}
	orows, err := s.db.QueryContext(ctx, `SELECT product_id,seller,rating,reviews,price_cents,stock,delivery FROM offers
		WHERE product_id = ANY(string_to_array($1, ',')::int[])`, strings.Join(ids, ","))
	if err != nil {
		return nil, err
	}
	defer orows.Close()
	for orows.Next() {
		var pid int
		var o catalog.Offer
		var price int64
		if err := orows.Scan(&pid, &o.Seller, &o.Rating, &o.Reviews, &price, &o.Stock, &o.Delivery); err != nil {
			return nil, err
		}
		o.Price = catalog.Money(price)
		ps[idx[pid]].Offers = append(ps[idx[pid]].Offers, o)
	}
	for i := range ps {
		ps[i].SortOffers()
	}
	return ps, orows.Err()
}

func (s *PGStore) Products(ctx context.Context) ([]catalog.Product, error) {
	return s.loadProducts(ctx, "")
}

func (s *PGStore) Product(ctx context.Context, slug string) (catalog.Product, error) {
	ps, err := s.loadProducts(ctx, "WHERE slug = $1", slug)
	if err != nil {
		return catalog.Product{}, err
	}
	if len(ps) == 0 {
		return catalog.Product{}, ErrNotFound
	}
	return ps[0], nil
}

func (s *PGStore) CreateUser(ctx context.Context, email, name, hash string) (User, error) {
	u := User{Email: strings.ToLower(strings.TrimSpace(email)), Name: name, PasswordHash: hash}
	err := s.db.QueryRowContext(ctx, `INSERT INTO users (email,name,password_hash) VALUES ($1,$2,$3) RETURNING id, created_at`,
		u.Email, name, hash).Scan(&u.ID, &u.CreatedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return User{}, ErrConflict
	}
	return u, err
}

func (s *PGStore) UserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := s.db.QueryRowContext(ctx, `SELECT id,email,name,password_hash,created_at FROM users WHERE email = $1`,
		strings.ToLower(strings.TrimSpace(email))).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (s *PGStore) PlaceOrder(ctx context.Context, userID int64, req []LineRequest, charge ChargeFunc) (Order, error) {
	lines, err := ValidateLines(req)
	if err != nil {
		return Order{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback()

	order := Order{UserID: userID, CreatedAt: s.now().UTC().Truncate(time.Microsecond)}
	for _, l := range lines {
		var it OrderItem
		var price int64
		var stock int
		err := tx.QueryRowContext(ctx, `SELECT p.id, p.slug, p.name, p.platform, p.region, o.seller, o.price_cents, o.stock
			FROM offers o JOIN products p ON p.id = o.product_id
			WHERE o.product_id = $1 AND o.seller = $2 FOR UPDATE OF o`, l.ProductID, l.Seller).
			Scan(&it.ProductID, &it.Slug, &it.Name, &it.Platform, &it.Region, &it.Seller, &price, &stock)
		if errors.Is(err, sql.ErrNoRows) {
			return Order{}, fmt.Errorf("%w: product %d is not sold by %s", ErrBadLine, l.ProductID, l.Seller)
		}
		if err != nil {
			return Order{}, err
		}
		if stock < l.Qty {
			return Order{}, fmt.Errorf("%w: %s by %s has %d left", ErrOutOfStock, it.Name, it.Seller, stock)
		}
		it.Price, it.Qty = catalog.Money(price), l.Qty
		order.Subtotal += it.Price * catalog.Money(l.Qty)
		order.Items = append(order.Items, it)
	}
	order.Fee = ServiceFee(order.Subtotal)
	order.Total = order.Subtotal + order.Fee

	if order.PaymentID, err = charge(order.Total); err != nil {
		return Order{}, err // rollback; rows were only locked
	}
	order.ID = NewOrderID(order.CreatedAt)
	for i, l := range lines {
		if _, err := tx.ExecContext(ctx, `UPDATE offers SET stock = stock - $3 WHERE product_id = $1 AND seller = $2`, l.ProductID, l.Seller, l.Qty); err != nil {
			return Order{}, err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE products SET sold = sold + $2 WHERE id = $1`, l.ProductID, l.Qty); err != nil {
			return Order{}, err
		}
		for range l.Qty {
			order.Items[i].Keys = append(order.Items[i].Keys, NewKey())
		}
	}
	items, _ := json.Marshal(order.Items)
	if _, err := tx.ExecContext(ctx, `INSERT INTO orders (id,user_id,created_at,subtotal_cents,fee_cents,total_cents,payment_id,items)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, order.ID, userID, order.CreatedAt, int64(order.Subtotal), int64(order.Fee), int64(order.Total), order.PaymentID, string(items)); err != nil {
		return Order{}, err
	}
	return order, tx.Commit()
}

func (s *PGStore) Orders(ctx context.Context, userID int64) ([]Order, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,user_id,created_at,subtotal_cents,fee_cents,total_cents,payment_id,items
		FROM orders WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Order{}
	for rows.Next() {
		var o Order
		var sub, fee, total int64
		var items string
		if err := rows.Scan(&o.ID, &o.UserID, &o.CreatedAt, &sub, &fee, &total, &o.PaymentID, &items); err != nil {
			return nil, err
		}
		o.Subtotal, o.Fee, o.Total = catalog.Money(sub), catalog.Money(fee), catalog.Money(total)
		if err := json.Unmarshal([]byte(items), &o.Items); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
