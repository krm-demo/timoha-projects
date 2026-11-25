# Deploy to Microsoft Azure

This runs the same Docker image as Render on **Azure Container Apps**, with **Azure Database for PostgreSQL – Flexible Server**. Both have free allowances, so a demo can cost €0.

## How much of our code changes?

**Go and JS code: none.** The app was built to be configured from the environment:

- `DATABASE_URL`, `JWT_SECRET`, `ALLOWED_ORIGINS` and `PORT` are already read in `backend/cmd/server/main.go`.
- `/api/health` already exists for health probes.
- The Postgres driver negotiates TLS, which Azure requires. Add `sslmode=require` to the URL anyway, so a misconfigured server is refused instead of silently falling back to an unencrypted connection.

| Change | Size | Status |
|---|---|---|
| `.github/workflows/image.yml`: publish the image to GitHub Container Registry (free) | ~35 lines YAML | done |
| This guide | docs | done |
| Optional: `infra/azure/main.bicep` (infrastructure as code instead of the CLI commands below) | ~100 lines | not done |
| Optional: GitHub Action that runs `az containerapp update --image …:${{ github.sha }}` on every push | ~30 lines YAML + a service principal secret | not done |

## Free allowances

- **Container Apps (Consumption plan):** the first 180,000 vCPU-seconds, 360,000 GiB-seconds and 2 million requests per subscription per month are free. Use `--min-replicas 0` so the app scales to zero when idle. Our Go server starts in under a second, so a cold start is barely noticeable.
- **PostgreSQL Flexible Server:** with an Azure free account it is free for 12 months, up to 750 h/month of Burstable **B1ms** with 32 GB storage and 32 GB backup. After that, B1ms costs about $12/month in the cheapest regions.

Check the current terms on azure.microsoft.com before you rely on them.

## Steps (Azure CLI)

Prerequisites:

- an Azure account;
- `az` installed and logged in (`az login`);
- the image published once by the "Publish container image" workflow.

On GitHub, open **Packages → the image → Package settings** and set its visibility to **Public**, so Azure can pull it without credentials.

```bash
RG=timoha-rg
LOC=westeurope
PG=timoha-pg-$RANDOM          # must be globally unique
PG_PASS='<a strong password>'
IMAGE=ghcr.io/<your-user>/<your-repo>:latest

az group create -n $RG -l $LOC

# 1. PostgreSQL (free-tier size). 0.0.0.0 = allow connections from Azure services only.
az postgres flexible-server create -g $RG -n $PG -l $LOC \
  --tier Burstable --sku-name Standard_B1ms --storage-size 32 --version 16 \
  --admin-user timoha --admin-password "$PG_PASS" --public-access 0.0.0.0
az postgres flexible-server db create -g $RG -s $PG -d timoha

# 2. Container Apps environment + the app
az extension add --name containerapp --upgrade
az containerapp env create -g $RG -n timoha-env -l $LOC
az containerapp create -g $RG -n timoha-market --environment timoha-env \
  --image $IMAGE --target-port 8080 --ingress external \
  --cpu 0.25 --memory 0.5Gi --min-replicas 0 --max-replicas 2 \
  --secrets db-url="postgres://timoha:$PG_PASS@$PG.postgres.database.azure.com:5432/timoha?sslmode=require" \
            jwt-secret="$(openssl rand -hex 32)" \
  --env-vars DATABASE_URL=secretref:db-url JWT_SECRET=secretref:jwt-secret \
             ALLOWED_ORIGINS=https://<your-user>.github.io

# 3. URL
az containerapp show -g $RG -n timoha-market --query properties.configuration.ingress.fqdn -o tsv
```

Open `https://<fqdn>`. The header should say **Live API**. On the first start the app creates its tables and seeds the catalog itself (`store/pgstore.go`).

To point GitHub Pages at it, set `TM_API_BASE` in `web/config.js` to `https://<fqdn>`.

Update after a new push: `az containerapp update -g $RG -n timoha-market --image $IMAGE`.

Delete everything: `az group delete -n $RG`.

## Which Azure database, and why

| Option | Use it when | For this project |
|---|---|---|
| **Flexible Server** | Default managed open-source Postgres, free tier, easy to move elsewhere | ✅ yes |
| **Elastic Clusters** (Citus on Flexible Server) | Data must be sharded across nodes (very large multi-tenant data) | overkill |
| **HorizonDB** (Aurora-like: separate compute and storage, zone-replicated, high write throughput) | Mission-critical, write-heavy, needs fast failover. Brand new (GA Nov 2026), so benchmark it first | overkill |
| Cosmos DB for PostgreSQL | Never for new projects: on a retirement path, being replaced by Elastic Clusters | ❌ |
| Self-hosted Postgres on a VM | Big steady load, strong in-house DBA, strict cost or latency needs | you own backups, HA and upgrades |

Interview line: *"The app only needs a Postgres URL. I deployed it on Flexible Server because it is plain Postgres with a free tier. If write throughput ever became the bottleneck, HorizonDB is a drop-in option. I avoided proprietary extensions so that stays possible."*
