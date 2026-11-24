# One image: Go API + the static frontend, served from the same origin.
FROM golang:1.24-alpine AS build
WORKDIR /src/backend
COPY backend/ .
# The Postgres driver is the only third-party dependency, and only for -tags postgres.
RUN go get github.com/jackc/pgx/v5@v5.7.2 && go mod tidy \
 && CGO_ENABLED=0 go build -tags postgres -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.20
RUN adduser -D -H app && mkdir -p /app/data && chown app /app/data
WORKDIR /app
COPY --from=build /out/server /app/server
COPY web /app/web
ENV PORT=8080 STATIC_DIR=/app/web DATA_FILE=/app/data/db.json
USER app
EXPOSE 8080
CMD ["/app/server"]
