# Roadmap

> Russian version: [ROADMAP-ru.md](ROADMAP-ru.md)

Ordered by what matters most for the submission. Estimates come in two forms:

- **by hand:** a person writing the code themselves;
- **with an agent:** an AI agent writes it and the person reviews it.

Either way Tymofii must be able to explain the result, so leave time to read it.

| # | Item | Why | By hand | With an agent |
|---|---|---|---|---|
| 1 | React + TypeScript frontend | **Required** by the assignment | 1–2 days | 1–2 h |
| 2 | Smart search with pills | **The differentiator** | 2–3 days | ~1 h |
| 3 | Public deployment (Render or Azure) | Reviewers need a live URL | 1–2 h | 15 min + your cloud login |
| 4 | English + German | Nice to have | ½–1 day | 15–30 min + a native-speaker review |
| 5 | Backend improvements | Depth, if time allows | varies | varies |

## 1. Port the frontend to React + TypeScript (required)

The backend API stays exactly as it is.

```
frontend/                      (Vite: npm create vite@latest frontend -- --template react-ts)
  src/api/client.ts            fetch wrapper: base URL, JSON, Bearer token, {error} → thrown Error
  src/api/types.ts             Product, Offer, Order, ListResult… (mirror backend/internal/catalog)
  src/auth/AuthContext.tsx     user + token, login/register/logout
  src/cart/CartContext.tsx     cart lines in localStorage (same shape as today)
  src/pages/Home.tsx           GET /api/home → render each block with its own useQuery
  src/pages/Store.tsx          filters ↔ URL search params, GET /api/products
  src/pages/Product.tsx        GET /api/products/:slug
  src/pages/Cart.tsx, Checkout.tsx, Order.tsx, Orders.tsx, Login.tsx, Register.tsx
  src/components/SearchBox.tsx debounced GET /api/search
  src/components/ProductCard.tsx, Header.tsx, Footer.tsx
  src/styles.css               start from web/styles.css (same colours and components)
```

- **Libraries:** `react-router-dom` for routes like `/product/:slug` and `/store?…`, and `@tanstack/react-query` for caching, loading and error states.
- **Protected routes:** `/checkout` and `/orders` redirect to `/login?next=…` when signed out.
- **Deploy:** either on Vercel with `VITE_API_BASE` set to the API URL, or build to `frontend/dist` and let the Go server serve it via `STATIC_DIR`. The second option gives one URL and no CORS.
- **The old frontend:** keep `web/` as the GitHub Pages preview that works without a backend, or delete it once React is live.
- **Tests:** Vitest + React Testing Library for SearchBox, the Store filters and the checkout error messages.

If smart search (item 2) is going to be built, build it **in React directly**, so the same UI isn't written twice.

## 2. The differentiator: smart search with pills

Full proposal: **[SMART_SEARCH.md](SMART_SEARCH.md)**. It covers the problem, the UX, the parser rules, the API contract, the code changes, a demo script, metrics, alternatives, risks and interview talking points.

In short: on key marketplaces the same game is sold as many near-identical listings (platform × region × edition × seller), and buyers end up with keys that don't activate.

- **Query → pills:** typing `elden ps5 under 30` turns the understood words into removable pills, `[PlayStation ✕] [≤ €30 ✕]`.
- **Grouped results:** one card per game, with its variants as pills.
- **"Works for you":** each variant shows whether it will activate in the buyer's country. The highlighted one is the cheapest key that *works*, not just the cheapest.

It is deterministic (no LLM), about 870 lines mostly in new files, and has 3 milestones that can each be shown on their own.

## 3. Public deployment

- **Render** (already configured): see [HANDOFF.md](HANDOFF.md), step 4.
- **Azure** (if the company prefers Microsoft): see [DEPLOY_AZURE.md](DEPLOY_AZURE.md). It uses Container Apps and PostgreSQL Flexible Server, both on free tiers, and needs no Go or JS code changes.

## 4. English + German

The approach and estimates are in the README ("Planned: English + German"). Do it in the React app with `react-i18next`, not in `web/`. Smart search already handles a few German and Russian words in queries (`unter 30`, `до 30`) at almost no cost.

## 5. Backend improvements (if time allows)

- **Search in the database.** For large catalogs, move filtering into SQL and use Postgres `pg_trgm` for typos. The smart search design stays the same.
- **Real payment provider in test mode.** Use Stripe PaymentIntents in test mode, confirmed by a webhook, instead of `internal/payments`.
- **Key inventory.** Add a `keys` table with pre-loaded codes per offer, so "stock" means the number of unsold keys. Hand out keys with `SELECT … FOR UPDATE SKIP LOCKED`.
- **Refresh tokens.** Use short-lived access tokens plus a refresh token stored on the server, which also makes "log out everywhere" possible.
- **OpenAPI spec and Swagger UI,** so reviewers can try the API in the browser.
- **Observability.** Add request IDs and a `/metrics` endpoint for Prometheus.
- **Seller side.** Sellers create and edit their own offers.

## 6. Other product ideas

- Wishlist and price alerts ("tell me when Elden Ring is under €15").
- Reviews for sellers.
- Price history per game, once the store has collected data over time.
