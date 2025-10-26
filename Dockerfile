# -------------------------------
# Stage 1: Build
# -------------------------------
FROM golang:1.23.0 AS builder

ENV GOPROXY=https://proxy.golang.org,direct
WORKDIR /src/WEB_UAKI

# Copy dependencies
COPY go.mod go.sum ./
RUN go mod download

# Install Buffalo CLI versi terbaru dan pastikan binary bisa diakses
RUN go install github.com/gobuffalo/cli/cmd/buffalo@latest \
    && cp $(go env GOPATH)/bin/buffalo /usr/local/bin/buffalo

# Copy source code
COPY . .
COPY database.yml ./database.yml

# Build binary aplikasi
RUN /usr/local/bin/buffalo build --static -o /bin/app


# -------------------------------
# Stage 2: Runtime
# -------------------------------
FROM debian:bookworm-slim

# Install dependensi dasar
RUN apt-get update && apt-get install -y bash ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary hasil build
COPY --from=builder /bin/app ./

# Copy Buffalo CLI agar bisa migrasi
COPY --from=builder /usr/local/bin/buffalo /usr/local/bin/buffalo

# Copy konfigurasi database
COPY --from=builder /src/WEB_UAKI/database.yml ./database.yml

ENV ADDR=0.0.0.0
EXPOSE 3000

# Jalankan aplikasi dulu, lalu migrasi setelah 5 detik
CMD ["bash", "-c", "./app & sleep 5 && buffalo pop migrate up || true && wait"]
