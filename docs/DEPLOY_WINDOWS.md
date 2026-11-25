# Running and deploying on Windows

The other docs show macOS/Linux (bash) commands. This page has the same steps for **Windows PowerShell**, which is what opens when you search for "PowerShell" in the Start menu.

Differences to keep in mind:

| bash (macOS/Linux) | PowerShell (Windows) |
|---|---|
| `cmd1 && cmd2` | Run them as **two separate lines**. `&&` only works in PowerShell 7+, not in the built-in Windows PowerShell 5.1 |
| `export NAME='value'` | `$env:NAME = 'value'`. It is valid only in that window, until you close it |
| `\` at the end of a line (continue the command) | `` ` `` (backtick) at the end of a line |
| `openssl rand -hex 32` | see "Generate a secret" below |

Use **single quotes** around URLs. They contain characters such as `?`, `=` and `&` that PowerShell would otherwise try to interpret.

## 1. Install the tools (one time)

```powershell
winget install --id Git.Git -e
winget install --id GoLang.Go -e
```

Close PowerShell and open it again, so the new programs are found. Then check:

```powershell
git --version
go version          # must say go1.24 or newer
```

## 2. Get the code

```powershell
cd $HOME
git clone https://github.com/krm-demo/timoha-projects.git
cd timoha-projects\backend
```

`pwd` must now end with `\timoha-projects\backend`.

## 3. Run locally with the built-in file database (no setup)

```powershell
go test ./...
go run ./cmd/server
```

Open http://localhost:8080. Stop the server with `Ctrl+C`.

## 4. Run locally against Neon Postgres

Add the Postgres driver (one time, two separate lines):

```powershell
go get github.com/jackc/pgx/v5@v5.7.2
go mod tidy
```

Set the connection string and start the server (same PowerShell window):

```powershell
$env:DATABASE_URL = 'postgresql://<user>:<password>@<host>.eu-central-1.aws.neon.tech/neondb?sslmode=require'
$env:JWT_SECRET   = '<a long random string, see below>'
go run -tags postgres ./cmd/server
```

Expected output includes `msg=database kind=postgres` and `msg=listening addr=:8080`.

`go get` and `go mod tidy` change `go.mod` and create `go.sum` locally. That's fine. Don't commit them unless you want the driver in every build; the Dockerfile, CI and Vercel add it themselves.

## Generate a secret (instead of `openssl rand -hex 32`)

Works in both Windows PowerShell 5.1 and PowerShell 7:

```powershell
$b = New-Object byte[] 32
[Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($b)
($b | ForEach-Object { $_.ToString('x2') }) -join ''
```

Copy the printed 64-character string into `JWT_SECRET`, here or in Vercel.

## 5. Optional: run the Docker image instead of Go

Needs Docker Desktop for Windows.

```powershell
docker run -p 8080:8080 `
  -e DATABASE_URL='postgresql://<user>:<password>@<host>/neondb?sslmode=require' `
  -e JWT_SECRET='<secret>' `
  ghcr.io/krm-demo/timoha-projects:latest
```

## 6. Deploy

- **Vercel + Neon:** everything happens in the browser, so nothing is Windows-specific. Follow [DEPLOY_VERCEL.md](DEPLOY_VERCEL.md) and use the PowerShell snippet above for `JWT_SECRET`.
- **Azure CLI** (if you use [DEPLOY_AZURE.md](DEPLOY_AZURE.md)): install it with `winget install --id Microsoft.AzureCLI -e`, then translate the bash variables and line breaks, for example:

  ```powershell
  $RG = 'timoha-rg'; $LOC = 'westeurope'; $PG = "timoha-pg-$(Get-Random -Maximum 99999)"
  az group create -n $RG -l $LOC
  az postgres flexible-server create -g $RG -n $PG -l $LOC `
    --tier Burstable --sku-name Standard_B1ms --storage-size 32 --version 16 `
    --admin-user timoha --admin-password '<strong password>' --public-access 0.0.0.0
  ```

## Troubleshooting

| Message | Fix |
|---|---|
| `go.mod file not found in current directory` | You're not in `timoha-projects\backend`: `cd $HOME\timoha-projects\backend` |
| `The token '&&' is not a valid statement separator` | Windows PowerShell 5.1: split the command into separate lines |
| `'go' is not recognized` | Open a new PowerShell window after installing Go |
| `bind: An attempt was made to access a socket…` / port in use | Another server is still running. Close it, or start on another port: `$env:PORT = '8081'` |
| Windows Firewall pop-up on first `go run` | Click **Allow** (private networks is enough). It's the local server on port 8080 |
