# timoha-projects — Timoha Market

A digital-keys marketplace in Eneba's style, built for the **Eneba Intern Assignment 2027**.

- **Frontend** (`web/`): a static SPA (vanilla JS, hash router). Preview: https://krm-demo.github.io/timoha-projects/
- **Backend** (`backend/`): a Go REST API with fuzzy search, homepage block services, auth, simulated checkout and orders.

The frontend checks `/api/health` when it loads. If the API answers, everything goes through the API and the header shows **Live API**. If not, as on GitHub Pages without a backend, it uses the static JSON in `web/data/` and keeps accounts and orders in the browser. The header then shows **Static demo**.

All data is demo data: prices and sellers are made up, and payments are simulated. The project is not affiliated with Eneba.

## Docs

| File | For |
|---|---|
| [docs/HANDOFF.md](docs/HANDOFF.md) | **Start here when you take over:** copy the repo, rename, enable Pages, deploy to Render, submission checklist |
| [docs/ROADMAP.md](docs/ROADMAP.md) · [RU](docs/ROADMAP-ru.md) | Next steps with estimates: React + TypeScript port (required), smart search, deployment, EN/DE, backend ideas |
| [docs/SMART_SEARCH.md](docs/SMART_SEARCH.md) · [RU](docs/SMART_SEARCH-ru.md) | **The proposed differentiator:** search that turns queries into pills, groups listings by game and shows which keys work in the buyer's country |
| [docs/DEPLOY_AZURE.md](docs/DEPLOY_AZURE.md) | Deploy to Azure Container Apps + PostgreSQL Flexible Server (free tiers), and which Azure database to choose |
| [docs/INTERVIEW_NOTES.md](docs/INTERVIEW_NOTES.md) | Likely interview questions, with answers pointing to the code |
| [docs/PROMPT_PLAYBOOK.md](docs/PROMPT_PLAYBOOK.md) | Ready-to-use, structured prompts for the next steps (React, smart search, deploy), to run in your own AI sessions |
| [docs/AI_SESSION_LOG.md](docs/AI_SESSION_LOG.md) | Summary of how this was built with an AI agent (the full log still needs to be exported) |

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

## How the frontend talks to the backend

```mermaid
sequenceDiagram
    participant B as Browser (web/)
    participant A as Go API (/api)
    participant D as DB (JSON file or PostgreSQL)
    B->>A: GET /api/health
    alt API answers
        Note over B: DS = ApiSource ("Live API")
    else no API (e.g. GitHub Pages only)
        Note over B: DS = StaticSource ("Static demo", data/*.json)
    end
    B->>A: GET /api/home
    A-->>B: [banner, categories, deals, bestsellers]
    par each block separately
        B->>A: GET /api/home/deals
        A->>D: products
        A-->>B: {service:"deals-service", products:[...]}
    end
    B->>A: GET /api/search?q=eldn rng (debounced, while typing)
    A-->>B: suggestions
    B->>A: POST /api/auth/login {email,password}
    A-->>B: {token (JWT), user}
    Note over B: token kept in localStorage,<br/>sent as Authorization: Bearer ...
    B->>A: POST /api/checkout {items:[{productId,seller,qty}], card}
    A->>D: BEGIN, lock offers, check stock
    A->>A: charge test card
    A->>D: decrement stock, sold += qty, save order + keys, COMMIT
    A-->>B: 201 {order with keys}
```

- **Transport:** plain REST over HTTPS with JSON bodies. Every response is either data or `{"error": "..."}` with a meaningful HTTP status, which the UI shows to the user as is.
- **Single entry point in the UI:** `web/data.js` exposes one interface (`DS.list`, `DS.product`, `DS.checkout`, …) with two implementations. `app.js` doesn't know whether it is talking to the API or to static files. This is why the GitHub Pages preview works without a server.
- **Same origin or cross-origin:** if the Go server serves `web/` itself, the frontend calls `/api/...` on the same origin. If the frontend stays on GitHub Pages and the API runs on Render, the browser makes cross-origin requests. The API allows them through CORS (`ALLOWED_ORIGINS=https://krm-demo.github.io`).
- **Stateless auth:** the server keeps no sessions. A signed JWT carries the user id and expiry. Any API instance can verify it with `JWT_SECRET`, so the API can be scaled horizontally.
- **Trust boundary:** the client sends only *what* to buy (product, seller, quantity). Prices, stock, totals and keys are all decided on the server inside one transaction.

## Why Go, and what the alternatives would look like

The assignment allows **PHP or Go** for the backend, so Java, Node.js and Python below are a thought experiment. They would not meet the requirement.

What Go gave us here (measured on this project):

- **Standard library only.** HTTP routing (Go 1.22+ `ServeMux` patterns), JSON, PBKDF2, HMAC/JWT, `database/sql` and the test HTTP server all come with the language. The only dependency is the Postgres driver, and only with `-tags postgres`. That leaves little supply-chain risk and nothing to update.
- **One small static binary:** about 6 MB, starts in under 0.3 s, about 12 MB RAM after serving requests. This matters on free hosting, where Render puts idle apps to sleep and gives them 512 MB. A fast cold start means a shorter wait for the first visitor.
- **Simple concurrency:** every request runs in its own goroutine with no async/await and no thread pools to tune. Checkout correctness comes from one DB transaction with row locks (or one mutex in the file store).
- **Static typing and fast builds:** the compiler catches most mistakes, `go test -race` finds data races, and a full build plus tests takes seconds.
- **Easy to read and explain.** That matters for the interview, where the code must be explained. There is no framework magic, so every route and every SQL statement is visible.

Raw speed is not the real reason here. For an I/O-bound API like this one (JSON in, database query, JSON out), Java, Node.js and Go are all far faster than needed. The differences show up in memory, startup time, deployment size and the amount of framework you depend on.

| | **Go** (chosen) | **Java** | **Node.js** | **Python** | **PHP** (the other allowed option) |
|---|---|---|---|---|---|
| Typical stack | stdlib `net/http` + `database/sql`/pgx | Spring Boot 3 (Web, Security, Data JPA) + Hibernate | NestJS or Fastify + Prisma/Drizzle, TypeScript | FastAPI + SQLAlchemy + Pydantic | Symfony or Laravel + Doctrine/Eloquent |
| Deployable | ~6 MB static binary | JAR on a JVM (~200 MB image); GraalVM native image possible | `node_modules` + runtime (~150 MB image) | interpreter + venv (~150 MB image) | PHP-FPM + nginx, or FrankenPHP |
| Idle memory | ~10–20 MB | ~150–300 MB (JVM) | ~50–80 MB | ~50–100 MB | ~20–40 MB per worker |
| Cold start | < 0.5 s | 3–10 s (≈0.1 s native) | ~0.5–1 s | ~1–2 s | per request (shared-nothing) |
| Concurrency | goroutines | threads; virtual threads since Java 21 | single-threaded event loop + async | asyncio (FastAPI) or worker processes | one process per request |
| Strengths | small, fast, explicit, few deps | most mature enterprise ecosystem, best tooling, transactions/security built in | same language as the React frontend (shared TS types), huge npm ecosystem | fastest to write, auto-generated OpenAPI docs, great for data/ML | Eneba's own main stack (likely closest to their codebase), very productive frameworks |
| Weaknesses | more hand-written code (no ORM/DI), smaller web ecosystem | heavy for a small service, slow start, lots of annotation "magic" | dependency sprawl, CPU-heavy work blocks the loop | slowest CPU-bound, GIL limits threads, typing is optional | no long-lived process by default, weaker typing than Go/Java |

What each version of *this* project would look like:

- **Java / Spring Boot.** `@RestController` classes for the same endpoints, JPA entities `Product`/`Offer`/`User`/`Order`, and Spring Security with a JWT filter. Checkout would be a `@Transactional` service method using `@Lock(PESSIMISTIC_WRITE)` on offers. You get strong guarantees and little hand-written plumbing, but also a heavy image and a slow start on Render's free tier.
- **Node.js / NestJS + Prisma.** Controllers and services in TypeScript. API types could be shared with the React frontend, which is the biggest plus. Checkout would use `prisma.$transaction` with `SELECT … FOR UPDATE` via raw SQL, and fuzzy search would use `pg_trgm` or a library such as Fuse.js. It is a natural choice for one full-stack TypeScript team.
- **Python / FastAPI.** Pydantic models give validation and a free Swagger UI. SQLAlchemy would handle the transaction, and `passlib`/`PyJWT` would handle auth. It is the quickest to prototype, but it needs more care with types and performance as the code grows.
- **PHP / Symfony.** API Platform or plain controllers, Doctrine entities with pessimistic locking, and LexikJWT for tokens. It is the option most aligned with Eneba's stack. Go was chosen to show a compact, dependency-free service that is easy to deploy.

## Planned: English + German

The GitHub Pages site is not translated yet. Estimates:

**1. Current site (vanilla JS): about half a day to one day by hand, or 15–30 minutes with an AI agent. The real bottleneck is a native speaker reviewing the German texts.**

- `web/i18n.js` with `en` and `de` dictionaries (about 80 UI strings) and a `t('key')` helper. `app.js` uses `t()` instead of literal text.
- A language switcher in the header. The language is picked from `?lang=` first, then the saved choice, then `navigator.language`, then English. It also sets `<html lang>`.
- Prices and dates use `Intl.NumberFormat` / `toLocaleString`, which formats correctly per locale (`€19.99` vs `19,99 €`).
- Content (product descriptions, banner texts): add `description_de` and friends to the demo data generator, with fallback to English.
- Backend: add `?lang=` / `Accept-Language` to `/api/home/*`, `/api/products*` and error messages, and return the localized fields.

**2. Production solution (after the React port): about two to three days, plus ongoing translation work**

- `react-i18next` (or FormatJS) with ICU messages for plurals and genders, and translation files lazy-loaded per language.
- The locale in the URL (`/de/store`, `/en/store`) with `hreflang` tags, for shareable links and SEO.
- Backend: a `product_translations(product_id, lang, name, description)` table, `Accept-Language` negotiation with fallback, and translated error codes. The API would return a code such as `PAYMENT_DECLINED` and the UI translates it.
- A translation workflow through a service such as Crowdin or Lokalise, plus a CI check that both languages have the same keys.
- The main ongoing cost is translating the content, not the code.

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
