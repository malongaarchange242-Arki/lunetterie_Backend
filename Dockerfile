# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Installer les dépendances de build
RUN apk add --no-cache git make

# Copier go.mod et go.sum
COPY go.mod go.sum* ./

# Télécharger les dépendances
RUN go mod download || true

# Copier le code source
COPY . .

# Compiler
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/api/main.go

# Runtime stage
FROM alpine:latest

WORKDIR /root/

# Installer les certificats SSL
RUN apk --no-cache add ca-certificates

# Copier le binaire depuis le builder
COPY --from=builder /app/main .

# Exposer le port
EXPOSE 8080

# Healthcheck
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Lancer l'application
CMD ["./main"]
