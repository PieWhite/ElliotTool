FROM node:24-alpine AS frontend-builder
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.26-alpine AS backend-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/wavesight .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S wavesight \
    && adduser -S -G wavesight wavesight
WORKDIR /app
COPY --from=backend-builder /out/wavesight /app/wavesight
COPY --from=frontend-builder /src/frontend/dist /app/frontend/dist
RUN mkdir -p /app/data && chown -R wavesight:wavesight /app
USER wavesight
ENV PORT=8080 \
    DATABASE_PATH=/app/data/wavesight.db \
    FRONTEND_DIR=/app/frontend/dist
EXPOSE 8080
VOLUME ["/app/data"]
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -q -O - http://127.0.0.1:8080/healthz >/dev/null || exit 1
ENTRYPOINT ["/app/wavesight"]
