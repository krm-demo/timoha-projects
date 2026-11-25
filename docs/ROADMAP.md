# Roadmap

Ordered by what matters most for the submission.

## 1. Port the frontend to React + TypeScript (required by the assignment)

Estimate: 1–2 days for a person writing it by hand. With an AI agent it takes about 1–2 hours, plus time to read and understand the result. The backend API stays exactly as it is.

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

## 1b. The differentiator: "Smart search" with pills

Every candidate builds a catalog, filters and a fuzzy search. To stand out, solve a real problem of key marketplaces in a way that can be shown in 10 seconds.

The problem: on Eneba-like stores, **the same game appears as many near-identical listings** (Steam / PSN / Xbox, EU / Global / US, Standard / Deluxe). Search results turn into a wall of duplicates, and the buyer still doesn't know which key works for them.

The feature, in three parts that share one UI element, the pill:

1. **Query → pills (intent parsing).** Typing `elden ps5 under 30` turns the recognised words into removable pills: `[PlayStation ✕] [≤ €30 ✕]`. Only `elden` stays as free text for the fuzzy search. The parser understands:
   - platform aliases: ps5/ps4/playstation → PSN, switch → Nintendo, xbox/xsx → Xbox, pc → Steam/GOG/…;
   - price: `under 30`, `<30`, `до 30`;
   - type: `gift card`, `dlc`;
   - region: `eu`, `global`.

   It is deterministic: no LLM, no cost, every rule is explainable in the interview.
2. **Grouped results.** One card per game, with **variant pills** such as `Steam · Global · €19.99`, `PSN · EU · €24.50` and `Xbox · US · €22.10`. The cheapest is highlighted, and clicking a pill opens that exact listing. The facet pills above the results show counts ("PlayStation 7") and act as instant filters.
3. **"Works for you" pill.** The buyer's country is taken from the browser locale, and they can change it. Each variant is marked ✓ *works in Spain* or ⚠ *US account only*. Region lock is the most common complaint about key stores, so this is an easy story to tell in an interview.

Size of the change to our code:

| Part | Where | Approx. lines |
|---|---|---|
| Intent parser + tests | `backend/internal/catalog/intent.go`, `intent_test.go` | ~150 + ~100 |
| Group by title (`Edition`/`Title` fields on Product) + tests | `catalog/group.go`, `catalog.go` | ~70 + ~60 |
| Region compatibility rule | `catalog/region.go` | ~40 |
| API: `/api/search` and `/api/products` return `pills` and `groups` | `api/api.go` | ~50 |
| Demo data: the same games on several platforms, regions and editions | `scripts/gen_data.py` | ~40 |
| UI: pills in the search box and store toolbar, grouped cards, country picker | `web/app.js`, `web/data.js` (static mirror), `web/styles.css` | ~200 |
| **Total** | | **≈ 700 lines, mostly new files; existing code changes only at the API and UI edges** |

Time: about an hour of agent work including tests, or 2–3 days for a person writing it by hand. Either way, Tymofii must be able to explain the parser and the grouping in the interview.

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
