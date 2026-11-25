# Deploy: Vercel (API) + Neon (database) + GitHub Pages (website)

All three are free and need no credit card.

```
Browser ── https://krm-demo.github.io/timoha-projects/  (GitHub Pages: web/)
   │  fetch /api/...  (CORS)
   ▼
https://timoha-api.vercel.app   (Vercel: backend/, Go runtime)
   │  postgres, TLS
   ▼
Neon Postgres, Frankfurt        (created via Vercel → Storage → Neon, Free plan)
```

## How it is wired

- `backend/vercel.json` tells Vercel to use the **Go preset** and to build with the Postgres driver:
  `go get pgx && go mod tidy && go build -tags postgres -o server ./cmd/server`.
  The Vercel build has internet access, so the driver is downloaded there.
- In Vercel, the project's **Root Directory** is `backend`, because the Go preset needs `go.mod` at the project root.
- The server reads `PORT`, `DATABASE_URL`, `JWT_SECRET` and `ALLOWED_ORIGINS` from environment variables. No code changes were needed.
- `web/config.js` points the GitHub Pages site at the Vercel API.

## Steps (one time)

1. **vercel.com → Add New… → Project → Import Git Repository.** Pick `timoha-projects`; connect GitHub if asked.
2. **Configure Project:**
   - Project Name: `timoha-api`. The URL becomes `https://timoha-api.vercel.app`. If you choose another name, update `web/config.js`.
   - Framework Preset: **Go**. If it isn't detected, `vercel.json` sets it anyway.
   - Root Directory: **Edit → `backend`**.
3. **Environment Variables** (add before clicking Deploy):

   | Name | Value |
   |---|---|
   | `DATABASE_URL` | Neon's **unpooled** URL (`postgresql://…@ep-….eu-central-1.aws.neon.tech/neondb?sslmode=require`, without `-pooler` and without `channel_binding`) |
   | `JWT_SECRET` | a long random string, e.g. the output of `openssl rand -hex 32`. **Required on Vercel:** several instances must share it, or logins break |
   | `ALLOWED_ORIGINS` | `https://krm-demo.github.io` |
   | `STATIC_DIR` | *(empty)*, because the website is served by GitHub Pages |

4. **Deploy.** The build takes about 1 minute. Then open `https://timoha-api.vercel.app/api/health`. It should print `{"ok":true,…}`.
5. Open the website: `https://krm-demo.github.io/timoha-projects/`. The header badge should say **Live API**.

Don't use Vercel's "Connect Project" button on the Neon database for this project. It would set `DATABASE_URL` to the *pooled* URL with `channel_binding=require`. Our server keeps its own connection pool, so the unpooled URL is the right one.

## Free-tier notes

- Vercel Hobby is free for personal, non-commercial projects. Instances start in about a second when idle.
- Neon Free: 1 GB storage and 100 CU-hours per month. The database sleeps after 5 minutes idle and wakes in about a second.
- Every push to `main` redeploys automatically.
