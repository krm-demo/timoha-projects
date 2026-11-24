package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/krm-demo/timoha-projects/backend/internal/auth"
	"github.com/krm-demo/timoha-projects/backend/internal/store"
)

type client struct {
	t     *testing.T
	srv   *httptest.Server
	token string
}

func newClient(t *testing.T) *client {
	st, err := store.OpenFile("")
	if err != nil {
		t.Fatal(err)
	}
	s := New(st, auth.Tokens{Secret: []byte("test")}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	s.Now = func() time.Time { return time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC) }
	srv := httptest.NewServer(CORS([]string{"*"}, s.Routes()))
	t.Cleanup(srv.Close)
	return &client{t: t, srv: srv}
}

func (c *client) do(method, path string, body any, wantStatus int, out any) {
	c.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, c.srv.URL+path, rd)
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != wantStatus {
		c.t.Fatalf("%s %s: status %d, want %d: %s", method, path, res.StatusCode, wantStatus, raw)
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			c.t.Fatalf("%s %s: decode: %v (%s)", method, path, err, raw)
		}
	}
}

type product struct {
	ID     int
	Slug   string
	Name   string
	Sold   int
	Offers []struct {
		Seller string
		Price  float64
		Stock  int
	}
}

func TestHomeBlocksComeFromServices(t *testing.T) {
	c := newClient(t)
	var layout struct {
		Blocks []struct{ ID, Endpoint string }
	}
	c.do("GET", "/api/home", nil, 200, &layout)
	if len(layout.Blocks) != 4 {
		t.Fatalf("layout: %+v", layout)
	}
	for _, b := range layout.Blocks {
		var block map[string]any
		c.do("GET", b.Endpoint, nil, 200, &block)
		if block["service"] == "" {
			t.Errorf("block %s has no service name", b.ID)
		}
	}
	var deals struct{ Products []product }
	c.do("GET", "/api/home/deals", nil, 200, &deals)
	if len(deals.Products) != 10 {
		t.Errorf("deals: %d products", len(deals.Products))
	}
	c.do("GET", "/api/home/nope", nil, 404, nil)
}

func TestCatalogAndSearch(t *testing.T) {
	c := newClient(t)
	var res struct {
		Items  []product
		Total  int
		Facets map[string][]struct {
			Value string
			Count int
		}
	}
	c.do("GET", "/api/products?platform=Steam&sort=price-asc&max=30", nil, 200, &res)
	if res.Total == 0 || len(res.Facets["platform"]) == 0 {
		t.Fatalf("list: %+v", res)
	}
	for i := 1; i < len(res.Items); i++ {
		if res.Items[i-1].Offers[0].Price > res.Items[i].Offers[0].Price {
			t.Fatal("not sorted by price")
		}
	}
	c.do("GET", "/api/products?min=abc", nil, 400, nil)

	var s struct{ Results []struct{ Name string } }
	c.do("GET", "/api/search?q=cyberpnk", nil, 200, &s)
	if len(s.Results) == 0 || s.Results[0].Name != "Cyberpunk 2077" {
		t.Fatalf("typo search: %+v", s)
	}

	var p struct {
		Product product
		Related []product
	}
	c.do("GET", "/api/products/elden-ring", nil, 200, &p)
	if p.Product.Name != "Elden Ring" || len(p.Product.Offers) < 3 || len(p.Related) == 0 {
		t.Fatalf("product page: %+v", p)
	}
	c.do("GET", "/api/products/nope", nil, 404, nil)
}

func TestAuthCheckoutAndOrders(t *testing.T) {
	c := newClient(t)
	c.do("GET", "/api/orders", nil, 401, nil)
	c.do("POST", "/api/checkout", map[string]any{}, 401, nil)

	reg := map[string]string{"name": "Timoha", "email": "tim@example.com", "password": "secret1"}
	c.do("POST", "/api/auth/register", map[string]string{"name": "x", "email": "bad", "password": "secret1"}, 400, nil)
	var sess struct {
		Token string
		User  struct{ Name string }
	}
	c.do("POST", "/api/auth/register", reg, 201, &sess)
	c.do("POST", "/api/auth/register", reg, 409, nil)
	c.do("POST", "/api/auth/login", map[string]string{"email": "tim@example.com", "password": "wrong!"}, 401, nil)
	c.do("POST", "/api/auth/login", map[string]string{"email": "TIM@example.com", "password": "secret1"}, 200, &sess)
	c.token = sess.Token
	var me struct{ User struct{ Name string } }
	c.do("GET", "/api/auth/me", nil, 200, &me)
	if me.User.Name != "Timoha" {
		t.Fatalf("me: %+v", me)
	}

	var p struct{ Product product }
	c.do("GET", "/api/products/hades-ii", nil, 200, &p)
	seller := p.Product.Offers[0].Seller
	items := []map[string]any{{"productId": p.Product.ID, "seller": seller, "qty": 2}}
	card := func(n string) map[string]string {
		return map[string]string{"number": n, "expiry": "12/29", "cvc": "123"}
	}

	c.do("POST", "/api/checkout", map[string]any{"items": items, "card": card("4000 0000 0000 0002")}, 402, nil)
	c.do("POST", "/api/checkout", map[string]any{"items": items, "card": card("1234 5678 9012 3456")}, 400, nil)
	c.do("POST", "/api/checkout", map[string]any{"items": []map[string]any{{"productId": p.Product.ID, "seller": "Nobody", "qty": 1}}, "card": card("4242 4242 4242 4242")}, 400, nil)

	var order struct {
		ID    string
		Total float64
		Items []struct {
			Keys  []string
			Price float64
		}
	}
	c.do("POST", "/api/checkout", map[string]any{"items": items, "card": card("4242 4242 4242 4242")}, 201, &order)
	if order.ID == "" || len(order.Items[0].Keys) != 2 || order.Items[0].Price != p.Product.Offers[0].Price {
		t.Fatalf("order: %+v", order)
	}

	var list struct{ Orders []struct{ ID string } }
	c.do("GET", "/api/orders", nil, 200, &list)
	if len(list.Orders) != 1 || list.Orders[0].ID != order.ID {
		t.Fatalf("orders: %+v", list)
	}
	c.do("GET", "/api/orders/"+order.ID, nil, 200, nil)

	var after struct{ Product product }
	c.do("GET", "/api/products/hades-ii", nil, 200, &after)
	if after.Product.Sold != p.Product.Sold+2 {
		t.Fatalf("sold %d -> %d", p.Product.Sold, after.Product.Sold)
	}

	c.token = "garbage"
	c.do("GET", "/api/orders", nil, 401, nil)
}
