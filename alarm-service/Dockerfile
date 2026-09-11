# syntax=docker/dockerfile:1

# ---------- Estágio 1: build ----------
FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

# Inclui o pacote docs/ gerado pelo swaggo, que já está versionado.
# Por isso não é preciso rodar "swag init" durante o build.
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/alarm-service .

# ---------- Estágio 2: runtime ----------
FROM alpine:3.20

# wget vem do busybox e é usado pelo healthcheck do compose.
RUN apk add --no-cache ca-certificates \
    && adduser -D -H -u 10001 appuser

COPY --from=builder /out/alarm-service /usr/local/bin/alarm-service

USER appuser

EXPOSE 8081

ENTRYPOINT ["/usr/local/bin/alarm-service"]