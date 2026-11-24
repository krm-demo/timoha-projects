# Interview notes: questions you will probably get

Each answer points to the code, so you can show it on screen.

**Walk me through a request: what happens when I search "eldn rng"?**
1. `SearchBox` (`web/app.js`, "header search") waits 150 ms after typing, then calls `DS.suggest`.
2. `web/data.js` → `GET /api/search?q=eldn%20rng`.
3. `backend/internal/api/api.go` `search()` loads the products, then calls `catalog.Search`.
4. `catalog/search.go` `Score()` normalizes both strings, then tries matches in this order: exact match → prefix → substring → per-word Levenshtein. A word may have 0, 1 or 2 typos depending on its length.
5. The results come back as JSON, and the UI ignores an answer if a newer request is already on its way (the `seq` counter).

**How is the homepage "not hardcoded"?**
`GET /api/home` returns the list of blocks. The UI then fetches each block from its own endpoint (`internal/home/home.go`). If one service fails, only that block disappears (the `.catch` in `pageHome`). Bestsellers change after real orders because `sold` is updated at checkout.

**How do you prevent a user from paying €1 for a €60 game?**
The client sends only `productId`, `seller` and `qty`. The price is read from the database inside the checkout transaction (`store/pgstore.go` `PlaceOrder`).

**What if two people buy the last key at the same time?**
In Postgres, `SELECT … FOR UPDATE OF o` locks the offer row, so the second buyer waits, then sees `stock = 0` and gets `409 not enough stock`. In the file store, one mutex serializes orders.

**What if the payment fails halfway?**
The charge happens *before* any write. If it fails, the transaction is rolled back, so stock and sold counts are untouched. This is tested in `store/contract_test.go` ("a failed payment must not change anything").

**How are passwords stored?**
With PBKDF2-SHA256, 210,000 iterations and a random 16-byte salt per user (`auth/auth.go`). They are compared in constant time, so the check doesn't leak timing. They are never stored in plain text.

**How does authentication work?**
Login returns a JWT signed with HMAC-SHA256 (`auth.Tokens`). The browser sends it as `Authorization: Bearer …`. `requireUser` middleware verifies the signature and expiry. There are no server sessions, so any instance can verify a token.

**Why is money an integer?**
`catalog.Money` is in cents (int64), because floats can't represent 0.1 exactly. It is converted to `19.99` only in JSON.

**Why two databases?**
`store.Store` is an interface. `FileStore` runs with zero setup, for tests, local dev and the demo. `PGStore` is for production. Both pass the same contract test, and CI runs it against real Postgres.

**Why Go and not PHP?**
Both are allowed. Go gives one small binary with almost no dependencies, a fast cold start on free hosting, and explicit code that is easy to explain. The full comparison is in the README ("Why Go").

**What would you do with more time?**
See `docs/ROADMAP.md`: the React port, i18n, `pg_trgm` search, Stripe test mode and a key inventory table.

**How did you use AI?**
Be honest and specific:

- what you asked;
- what the agent produced;
- what you checked, changed or rejected.

Eneba explicitly wants to see this. Read the code until you could rewrite each part yourself.
