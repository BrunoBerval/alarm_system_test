# syntax=docker/dockerfile:1

# ---------- Estágio 1: build ----------
FROM golang:1.25-alpine AS builder

WORKDIR /src

# Copiado separadamente para aproveitar o cache de camadas:
# só refaz o download se go.mod/go.sum mudarem.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 gera um binário estático, que roda em qualquer base sem libc.
# -trimpath remove os caminhos absolutos da máquina de build.
# -s -w removem tabela de símbolos e DWARF (binário menor).
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/event-service .

# ---------- Estágio 2: runtime ----------
FROM alpine:3.20

# ca-certificates fica pronto caso o broker passe a usar TLS (mqtts://).
RUN apk add --no-cache ca-certificates \
    && adduser -D -H -u 10001 appuser

COPY --from=builder /out/event-service /usr/local/bin/event-service

# Não roda como root.
USER appuser

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/event-service"]