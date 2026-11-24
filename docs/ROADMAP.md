# Roadmap

Ordered by what matters most for the submission.

## 1. Port the frontend to React + TypeScript (required by the assignment)

Estimate: 1–2 days. The backend API stays exactly as it is.

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

- Libraries: `react-router-dom` (routes like `/product/:slug`, `/store?…`) and `@tanstack/react-query` (caching, loading and error states).
- Protected routes: `/checkout` and `/orders` redirect to `/login?next=…` when signed out.
- Deploy on Vercel. Set `VITE_API_BASE` to the Render URL, and add the Vercel domain to `ALLOWED_ORIGINS` on Render. Alternatively, build to `frontend/dist` and let the Go server serve it with `STATIC_DIR`. That gives one URL with no CORS.
- Keep `web/` as the no-backend GitHub Pages preview, or delete it once the React app is deployed.
- Tests: Vitest + React Testing Library for SearchBox, Store filters and the checkout error messages.

## 2. English + German

The estimate and approach are in the README ("Planned: English + German"). Do it in the React app with `react-i18next` rather than in `web/`.

## 3. Backend improvements (nice to have)

- **Search in the database.** For large catalogs, move filtering into SQL and use Postgres `pg_trgm` (`similarity(name, $1) > 0.3`) for typos.
- **Real payment provider test mode.** Use Stripe Checkout or PaymentIntents in test mode instead of `internal/payments`, confirmed by a webhook.
- **Key inventory.** Add a `keys` table with pre-loaded codes per offer, so stock means the number of unsold keys. Hand out a key with `SELECT … FOR UPDATE SKIP LOCKED`.
- **Refresh tokens / logout everywhere.** Use short-lived access tokens plus a refresh token stored server-side.
- **OpenAPI spec** (`docs/openapi.yaml`) and Swagger UI, so reviewers can try the API.
- **Observability.** Add request IDs and a `/metrics` endpoint for Prometheus.
- **Seller side.** Sellers create and edit their own offers. This extends the marketplace idea.

## 4. Product ideas

- Wishlist and price alerts ("notify me when Elden Ring is under €15").
- Reviews for sellers.
- Region checker: warn when a key's region doesn't match the user's country.
