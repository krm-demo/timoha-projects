# ai-timoha-projects — Timoha Market (prototype)

Clickable prototype of the **Eneba Intern Assignment 2027**: a digital-keys marketplace in Eneba's style.
Static site (vanilla JS, hash router) — deployed to GitHub Pages as a quick preview of what the final app should look like.

## What works
| Requirement | Prototype |
|---|---|
| Store | Catalog with filters (platform, region, type, genre, price), sorting, URL-synced state |
| Homepage | Built from `data/home/layout.json`; each block (banner, categories, deals, bestsellers) loads from its own "service" JSON — nothing hardcoded in the UI |
| Search | Header search from any page, live suggestions, typo- and partial-name tolerant (Levenshtein per word), e.g. `eldn rng` → Elden Ring |
| Product page | Details + offers from several sellers, sorted by price, seller rating & stock |
| Checkout | Cart → simulated payment (test card `4242 4242 4242 4242`, `4000 0000 0000 0002` declines) → digital key issued |
| Auth | Register / sign in (demo: localStorage, SHA-256 password hash); checkout and order history require sign-in |
| Styling | Purple gradient, yellow CTAs, white product cards, Inter font |

All data is demo data; prices and sellers are made up. Not affiliated with Eneba.

## Plan for the real build (per the assignment)
- **Frontend:** React + TypeScript (Vite), React Router, TanStack Query.
- **Backend:** PHP (Symfony/Laravel) or Go, REST: `/api/home/{block}`, `/api/products`, `/api/search`, `/api/products/{slug}/offers`, `/api/cart`, `/api/checkout`, `/api/auth/*`, `/api/orders`.
- **DB:** PostgreSQL (`products`, `sellers`, `offers`, `users`, `orders`, `order_items`, `keys`); search via `pg_trgm` for typo tolerance.
- **Payments:** Stripe test mode.
- **Deploy:** Render / Vercel free tier.

## Run locally
```
python3 -m http.server 8000
```
