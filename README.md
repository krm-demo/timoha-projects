# timoha-projects — Timoha Market

A digital-keys marketplace in Eneba's style, built for the **Eneba Intern Assignment 2027**.

- **Frontend** (`web/`): a static SPA (vanilla JS, hash router). Preview: https://krm-demo.github.io/timoha-projects/
- **Backend** (`backend/`): a Go REST API with fuzzy search, homepage block services, auth, simulated checkout and orders.

The frontend checks `/api/health` when it loads. If the API answers, everything goes through the API and the header shows **Live API**. If not, as on GitHub Pages without a backend, it uses the static JSON in `web/data/` and keeps accounts and orders in the browser. The header then shows **Static demo**.

All data is demo data: prices and sellers are made up, and payments are simulated. The project is not affiliated with Eneba.

## Requirements → implementation

| Requirement | Where |
|---|---|
| **Store**: browse, filter by platform/region/type/genre/price, sort, search | `GET /api/products`, `internal/catalog/query.go`. Facet counts exclude their own filter, so the sidebar shows what each extra filter would return |
| **Homepage**: blocks with their own data sources, nothing hardcoded | `GET /api/home` returns the layout. `GET /api/home/{banner,categories,deals,bestsellers}` are separate services in `internal/home`: CMS banner, in-stock platforms, top discounts, top sellers (updated by real orders) |
| **Search**: from anywhere, typo- and partial-name tolerant | `GET /api/search`, `internal/catalog/search.go`: exact > prefix > substring > per-word Levenshtein, with 0/1/2 typos allowed by word length. `eldn rng` → Elden Ring, `cyberpnk` → Cyberpunk 2077 |
| **Product page**: details and offers from several sellers | `GET /api/products/{slug}`: offers sorted with in-stock first, then cheapest; includes rating, reviews and stock, plus related products |
| **Checkout**: cart → pay → receive key | `POST /api/checkout`: prices are read on the server, never taken from the client. One transaction checks stock, charges, decrements stock, increments sold and issues keys. A failed payment changes nothing |
| **Authentication**: sign up and sign in; checkout and orders need sign-in | `POST /api/auth/register`, `/login`, `GET /api/auth/me`. Passwords use PBKDF2-SHA256 (210k iterations), sessions use HS256 JWT, and auth endpoints are rate-limited per IP |
| **Eneba styling** | Purple gradient, yellow CTAs, white product cards, Inter |

Test cards: `4242 4242 4242 4242` succeeds, `4000 0000 0000 0002` is declined, and `4000 0000 0000 9995` fails with insufficient funds. Use any future expiry and any CVC.

## Backend design

```
backend/
  cmd/server        config from env, graceful shutdown, serves API + web/
  internal/catalog  domain model, Money (cents), filtering/facets/sorting, fuzzy search
  internal/home     one service per homepage block + layout registry
  internal/auth     PBKDF2 password hashing, JWT issue/verify
  internal/payments test-mode card processor (Luhn, expiry, CVC, Stripe-like test cards)
  internal/store    Store interface + two databases:
                      FileStore  – embedded JSON-file DB (default, zero deps)
                      PGStore    – PostgreSQL (-tags postgres), row locks at checkout
  internal/api      handlers, CORS, logging/recover, rate limiting
```

- Only the standard library is used (Go 1.24). The Postgres driver (`pgx`) is the one third-party dependency, and only builds with `-tags postgres`.
- Both databases pass the same contract test (`internal/store/contract_test.go`). CI runs it against a real PostgreSQL 16.
- The catalog is small, so filtering and fuzzy search run in Go over the product list. With a larger catalog this would move into SQL, for example with `pg_trgm` for typo tolerance.

### API

| Method | Path | Auth |
|---|---|---|
| GET | `/api/health` | |
| GET | `/api/home`, `/api/home/{block}` | |
| GET | `/api/products?q&platform&region&type&genre&min&max&sort&page&limit` | |
| GET | `/api/products/{slug}` | |
| GET | `/api/search?q&limit` | |
| POST | `/api/auth/register` `{name,email,password}` → `{token,user}` | |
| POST | `/api/auth/login` `{email,password}` → `{token,user}` | |
| GET | `/api/auth/me` | Bearer |
| POST | `/api/checkout` `{items:[{productId,seller,qty}], card:{number,expiry,cvc}}` | Bearer |
| GET | `/api/orders`, `/api/orders/{id}` | Bearer |

Errors are returned as `{"error": "..."}` with the matching status: 400 for bad input, 401 when not signed in, 402 for a declined payment, 404 when not found, 409 for out of stock or a duplicate email, and 429 when rate-limited.

## Run locally

```bash
cd backend
go test ./...                     # unit + API tests
go run ./cmd/server               # http://localhost:8080  (API + frontend, JSON-file DB in data/db.json)

# with PostgreSQL
go get github.com/jackc/pgx/v5@v5.7.2 && go mod tidy
DATABASE_URL=postgres://user:pass@localhost:5432/timoha go run -tags postgres ./cmd/server
```

Environment variables: `PORT`, `DATABASE_URL`, `DATA_FILE`, `JWT_SECRET`, `ALLOWED_ORIGINS`, `STATIC_DIR`. See `cmd/server/main.go`.

Frontend only: `cd web && python3 -m http.server`. To use a backend running elsewhere, open the site with `?api=https://your-api`, or set `TM_API_BASE` in `web/config.js`.

## Deploy (free tier)

- **Render**: New → Blueprint → this repo (`render.yaml`). It builds the `Dockerfile`, creates a free Postgres database and generates `JWT_SECRET`. The whole app (API and frontend) then runs at the Render URL. To make GitHub Pages use the same API, set `TM_API_BASE` in `web/config.js` to the Render URL.
- **GitHub Pages**: `.github/workflows/pages.yml` publishes `web/` on every push to `main`.

Free-tier caveats: Render's free web services sleep when idle, so the first request takes about 30–60 s. Its free Postgres expires after 30 days. Without `DATABASE_URL`, the JSON-file DB is lost on each redeploy.

## Not done yet (vs. the assignment)

- The frontend is vanilla JS. The assignment asks for **React (TypeScript preferred)**, so the next step is to port `web/` to Vite + React + TS on top of the same API.
- The agent chat history required for submission still has to be exported to a `.md` file.

## Data

`scripts/gen_data.py` generates the demo catalog into `web/data/` and the backend seeds (`backend/internal/store/seed`, `backend/internal/home/banner.json`).
