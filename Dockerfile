# =========================
# Stage 1: Build Buffalo App
# =========================
FROM golang:1.23.0 AS builder

# Set working directory
WORKDIR /src/WEB_UAKI

# Set Go proxy (optional tapi disarankan)
ENV GOPROXY=https://proxy.golang.org,direct

# Copy go.mod & go.sum, lalu download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Install Buffalo CLI dan Pop CLI (soda)
RUN go install github.com/gobuffalo/cli/cmd/buffalo@latest
RUN go install github.com/gobuffalo/pop/v6/soda@latest

# Copy seluruh source code
COPY . .

# Build binary Buffalo app
RUN buffalo build -o /bin/app

# =========================
# Stage 2: Runtime Container
# =========================
FROM debian:bookworm-slim

# Install dependencies minimal
RUN apt-get update && apt-get install -y ca-certificates bash curl && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy hasil build dan soda dari stage builder
COPY --from=builder /bin/app /app/app
COPY --from=builder /go/bin/soda /usr/local/bin/soda

# Copy file konfigurasi database
COPY database.yml ./database.yml

# Expose port aplikasi
EXPOSE 8080

# Jalankan aplikasi dan migrasi setelahnya
CMD ["bash", "-c", "./app & sleep 5 && soda migrate up || true && wait"]
