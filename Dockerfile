FROM node:22-bookworm-slim AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.26.3-bookworm AS backend-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY web/ ./web/
COPY .env.example ./.env.example
COPY --from=frontend-builder /app/frontend/dist /app/web/dist
RUN go build -o /out/classical-chinese-quiz ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=backend-builder /out/classical-chinese-quiz /app/classical-chinese-quiz
RUN mkdir -p /data
ENV APP_ENV=production \
    HTTP_ADDR=:8080 \
    DB_PATH=/data/app.db
EXPOSE 8080
ENTRYPOINT ["/app/classical-chinese-quiz"]
