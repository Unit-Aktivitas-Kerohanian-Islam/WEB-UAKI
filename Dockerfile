# =========================
# Stage 1: Build Buffalo App
# =========================
FROM golang:1.23.0 AS builder

# Set Go proxy
ENV GOPROXY=https://proxy.golang.org,direct

# Set working directory di dalam container
WORKDIR /src/WEB_UAKI

# Copy file dependensi terlebih dahulu
COPY go.mod go.sum ./
RUN go mod download

# Install Buffalo CLI dan tool migrasi Soda
RUN go install github.com/gobuffalo/cli/cmd/buffalo@latest
RUN go install github.com/gobuffalo/pop/v6/soda@latest

# Copy seluruh source code, termasuk folder migrations
COPY . .
COPY migrations ./migrations

# Build aplikasi Buffalo menjadi binary
RUN buffalo build -o /bin/app


# =========================
# Stage 2: Runtime Container
# =========================
FROM debian:bookworm-slim

# Install dependensi minimal + PostgreSQL client agar pg_dump tersedia
RUN apt-get update && apt-get install -y ca-certificates bash curl postgresql-client && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy binary hasil build & tool soda
COPY --from=builder /bin/app /app/app
COPY --from=builder /go/bin/soda /usr/local/bin/soda

# Copy file konfigurasi database jika ada
COPY database.yml ./database.yml

# Set environment
ENV ADDR=0.0.0.0
ENV GO_ENV=production

# Port default Buffalo
EXPOSE 8080

# Jalankan aplikasi + migrasi otomatis
# (tunggu 5 detik agar DB siap, lalu jalankan migrasi tanpa dump schema)
CMD ["bash", "-c", "./app & sleep 5 && soda migrate up --no-dump-schema || true && wait"]
