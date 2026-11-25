# Handoff: making this project yours

Hi Tymofii! This repo is a working starting point for the **Eneba Intern Assignment 2027**. It contains:

- a clickable frontend in `web/`;
- a Go backend with tests in `backend/`;
- CI, a Dockerfile and a Render deploy config.

This page explains how to take it over and what still needs to be done before you submit.

## 1. Copy it into your own GitHub

Option A: fork on GitHub. This keeps the history. Click **Fork** on `krm-demo/timoha-projects`.

Option B: a clean repo with your own history. The submission is yours, so this one is recommended.

```bash
git clone https://github.com/krm-demo/timoha-projects.git eneba-market
cd eneba-market
rm -rf .git && git init -b main
git add -A && git commit -m "Initial import"
# create an EMPTY public repo on github.com (no README), then:
git remote add origin https://github.com/<your-user>/<your-repo>.git
git push -u origin main
```

## 2. Replace our names and URLs with yours

| File | What to change |
|---|---|
| `README.md` | `krm-demo/timoha-projects` and `krm-demo.github.io/timoha-projects` → your repo / Pages URL |
| `web/index.html` | footer link `id="repoLink"` |
| `render.yaml` | `ALLOWED_ORIGINS` → `https://<your-user>.github.io` |
| `web/config.js` | `TM_API_BASE` → your Render URL once the API is deployed |
| `backend/go.mod` | module path, optional: `github.com/<your-user>/<your-repo>/backend`, then update imports with `grep -rl krm-demo backend \| xargs sed -i 's#krm-demo/timoha-projects#<your-user>/<your-repo>#g'` |

Quick check that nothing is left: `grep -rn "krm-demo" .`

## 3. Turn on GitHub Pages and CI

1. Repo **Settings → Pages → Source: GitHub Actions**. Without this the first Pages run fails with *"Create Pages site failed"*.
2. **Actions** tab → re-run "Deploy to GitHub Pages". The site will be at `https://<your-user>.github.io/<your-repo>/`.
3. The **CI** workflow runs on every push:
   - Go formatting, vet and tests;
   - the database tests against a real PostgreSQL;
   - a Docker build plus a smoke test.

   Keep it green.

## 4. Deploy the backend (free)

1. Sign up at render.com with GitHub.
2. **New → Blueprint** → choose your repo. Render reads `render.yaml`. It builds the Docker image, creates a free Postgres database and generates `JWT_SECRET`.
3. When it's live, open `https://<service>.onrender.com`. The full app is served there, and the header badge says **Live API**.
4. Optional: point GitHub Pages at it. Set `TM_API_BASE` in `web/config.js` to the Render URL, then commit.

Free-tier notes:

- The service sleeps after inactivity, so the first request takes 30–60 s. Open it once before you send the link.
- The free Postgres expires after 30 days.

## 5. Run it on your computer

You need Go 1.24 or newer.

```bash
cd backend
go test ./...          # all tests
go run ./cmd/server    # → http://localhost:8080 (API + website)
```

Test cards:

- `4242 4242 4242 4242` succeeds;
- `4000 0000 0000 0002` is declined;
- `4000 0000 0000 9995` fails with insufficient funds.

## 6. Submission checklist

The deadline is **7 days after you received the assignment**. The email from Eneba arrived on 19 Nov 2026, so plan for **26 Nov 2026** at the latest. Send everything to **internship@eneba.com**:

- [ ] Your full name
- [ ] CV in English
- [ ] GitHub link(s) to the project
- [ ] Public URL of the deployed app (Render, or Vercel for a React frontend)
- [ ] Full AI chat history as a `.md` file or a link. Export the whole conversation; a summary is not enough. See `docs/AI_SESSION_LOG.md`.
- [ ] Optional but recommended: a short note on scope and decisions. You can adapt "Why Go" and the requirements table from the README.

Gaps against the assignment, by priority:

1. **The frontend must be React (TypeScript preferred).** Today it is vanilla JS. The plan is in `docs/ROADMAP.md`.
2. **To stand out:** the smart search proposal in `docs/SMART_SEARCH.md` (Russian: `SMART_SEARCH-ru.md`). Before pitching it, check eneba.com for the problem it solves.
3. Everything else in the assignment is covered: store, homepage blocks from separate services, typo-tolerant search, product page with seller offers, checkout with keys, auth, Eneba-like styling.

## 7. Be ready to explain the code

The assignment says *"You must be able to explain your code."* Read `docs/INTERVIEW_NOTES.md`. It lists the questions you are likely to get and where the answers are in the code.
