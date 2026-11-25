# Handoff for Windows: making this project yours

> The same as [HANDOFF.md](HANDOFF.md), with every command written for **Windows PowerShell**. Open it from the Start menu: type "PowerShell".
> In PowerShell, run each command on its own line: `&&` doesn't work in the built-in Windows PowerShell 5.1. Use `$env:NAME = 'value'` instead of `export`, and single quotes around URLs.

Hi Tymofii! This repo is a working starting point for the **Eneba Intern Assignment 2027**:

- a clickable frontend in `web/`, published on GitHub Pages;
- a Go backend with tests in `backend/`;
- a live setup: **Go API on Vercel + Neon Postgres + website on GitHub Pages**, all free.

## 0. Install the tools (one time)

```powershell
winget install --id Git.Git -e
winget install --id GoLang.Go -e
```

Close PowerShell, open a new window, then check:

```powershell
git --version
go version          # go1.24 or newer
```

## 1. Copy it into your own GitHub

Option A: fork on GitHub. This keeps the history. Click **Fork** on `krm-demo/timoha-projects`.

Option B: a clean repo with your own history. The submission is yours, so this one is recommended.

1. On github.com, create an **empty public** repository (no README, no .gitignore), for example `eneba-market`.
2. In PowerShell:

```powershell
cd $HOME
git clone https://github.com/krm-demo/timoha-projects.git eneba-market
cd eneba-market
Remove-Item -Recurse -Force .git
git init -b main
git add -A
git commit -m "Initial import"
git remote add origin https://github.com/<your-user>/eneba-market.git
git push -u origin main
```

If `git commit` asks who you are, run these once, then repeat the commit:

```powershell
git config --global user.name  "Tymofii Svyrhun"
git config --global user.email "tymofiisvyrhun@gmail.com"
```

## 2. Replace our names and URLs with yours

| File | What to change |
|---|---|
| `README.md`, `docs\*.md` | `krm-demo/timoha-projects` and `krm-demo.github.io/timoha-projects` → your repo and Pages URL |
| `web\index.html` | the footer link `id="repoLink"` |
| `web\config.js` | the Vercel API URL (see step 4) |
| `render.yaml` | `ALLOWED_ORIGINS`, only if you use Render |

Find what's left, using Git's own search (works the same on Windows):

```powershell
git grep -n "krm-demo"
```

Optional: rename the Go module path in all Go files (UTF-8 without BOM, so Go stays happy):

```powershell
$old = 'krm-demo/timoha-projects'; $new = '<your-user>/eneba-market'
Get-ChildItem backend -Recurse -Include *.go, go.mod | ForEach-Object {
  $text = [IO.File]::ReadAllText($_.FullName)
  if ($text.Contains($old)) { [IO.File]::WriteAllText($_.FullName, $text.Replace($old, $new)) }
}
cd backend
go build ./...
cd ..
```

## 3. Turn on GitHub Pages and CI

1. Your repo → **Settings → Pages → Source: GitHub Actions**. Without it, the first Pages run fails with *"Create Pages site failed"*.
2. **Actions** tab → "Deploy to GitHub Pages" → **Re-run jobs**. The site will be at `https://<your-user>.github.io/eneba-market/`.
3. **CI** runs on every push: Go tests, database tests on a real PostgreSQL, and a Docker build. Keep it green.

If a Pages run fails with *"Fetching artifact metadata failed"*, that's a temporary GitHub problem. Re-run it later.

## 4. Deploy the backend: Vercel + Neon (free, no card)

Full guide: [DEPLOY_VERCEL.md](DEPLOY_VERCEL.md). In short:

1. **Database:** vercel.com → **Storage → Create Database → Neon**, plan **Free**, region **Frankfurt**. Skip "Connect a Project". On the database page, copy `DATABASE_URL_UNPOOLED`, the address *without* `-pooler`.
2. **Secret for logins:** generate one in PowerShell:

   ```powershell
   $b = New-Object byte[] 32
   [Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($b)
   ($b | ForEach-Object { $_.ToString('x2') }) -join ''
   ```

3. **API:** vercel.com → **Add New… → Project** → import your repo.
   - Project Name: `<something>-api`
   - **Root Directory: `backend`**
   - Environment Variables:

     | Name | Value |
     |---|---|
     | `DATABASE_URL` | the unpooled Neon URL |
     | `JWT_SECRET` | the string from step 2 |
     | `ALLOWED_ORIGINS` | `https://<your-user>.github.io` |
     | `STATIC_DIR` | *(empty)* |

   → **Deploy**. Check `https://<something>-api.vercel.app/api/health`: it should print `{"ok":true,…}`.
4. **Point the website at it:** edit `web\config.js` and put your Vercel URL in place of `https://timoha-api.vercel.app`. Then:

   ```powershell
   git add web/config.js
   git commit -m "Point the site at my Vercel API"
   git push
   ```

   After the Pages deploy, the site's header badge says **Live API**.

From then on, every push to `main` redeploys the API on Vercel automatically, and the website too if `web\` changed.

## 5. Run it on your computer

```powershell
cd $HOME\eneba-market\backend
go test ./...
go run ./cmd/server
```

Open http://localhost:8080. This uses a local file database, so no setup is needed. Stop the server with `Ctrl+C`. If Windows Firewall asks, click **Allow**.

Against your Neon database instead:

```powershell
go get github.com/jackc/pgx/v5@v5.7.2
go mod tidy
$env:DATABASE_URL = '<the unpooled Neon URL>'
$env:JWT_SECRET   = '<your secret>'
go run -tags postgres ./cmd/server
```

Before committing, undo the local driver change so `go.mod` stays clean:

```powershell
git checkout -- go.mod
Remove-Item go.sum -ErrorAction SilentlyContinue
```

Test cards:

- `4242 4242 4242 4242` succeeds;
- `4000 0000 0000 0002` is declined;
- `4000 0000 0000 9995` fails with insufficient funds.

More Windows details and fixes for common errors: [DEPLOY_WINDOWS.md](DEPLOY_WINDOWS.md).

## 6. Submission checklist

Deadline: **7 days after you received the assignment**. The email arrived on 19 Nov 2026, so **26 Nov 2026** at the latest. Send to **internship@eneba.com**:

- [ ] Your full name
- [ ] CV in English
- [ ] GitHub link to your repo
- [ ] Public URL of the app: your GitHub Pages site (with **Live API**) and/or the Vercel URL
- [ ] Full AI chat history as a `.md` file or a link. Export the whole conversation; a summary is not enough. See [AI_SESSION_LOG.md](AI_SESSION_LOG.md) and [PROMPT_PLAYBOOK.md](PROMPT_PLAYBOOK.md).
- [ ] Optional: a short note on scope and decisions. You can adapt "Why Go" and the requirements table from the README.

Gaps against the assignment:

1. **The frontend must be React (TypeScript preferred).** Today it is vanilla JS. The plan is in [ROADMAP.md](ROADMAP.md).
2. **To stand out:** the smart search proposal in [SMART_SEARCH.md](SMART_SEARCH.md).
3. Everything else is covered: store, homepage blocks from separate services, typo-tolerant search, product page with seller offers, checkout with keys, auth, Eneba-like styling.

## 7. Be ready to explain the code

*"You must be able to explain your code."* Read [INTERVIEW_NOTES.md](INTERVIEW_NOTES.md) and the comments in `backend\internal\...`.
