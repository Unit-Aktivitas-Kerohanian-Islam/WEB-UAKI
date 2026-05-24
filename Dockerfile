# -------------------------------
# Stage 1: Build (pakai Alpine)
# -------------------------------
FROM golang:1.24-alpine AS builder

# Install tools dasar
RUN apk add --no-cache git bash build-base

# Gunakan Go proxy
ENV GOPROXY=https://proxy.golang.org,direct

# Set working directory di dalam container
WORKDIR /src/WEB_UAKI

# Copy file dependensi terlebih dahulu
COPY go.mod go.sum ./
RUN go mod download

# Install Buffalo CLI dan Soda CLI
RUN go install github.com/gobuffalo/cli/cmd/buffalo@v0.18.14 && \
    go install github.com/gobuffalo/pop/v6/soda@v6.1.1

# Copy seluruh source code project
COPY . .

# Pastikan file konfigurasi ikut disalin
COPY database.yml ./database.yml

# Jalankan proses build Buffalo
RUN /go/bin/buffalo build --static -o /bin/app


# -------------------------------
# Stage 2: Runtime (masih Alpine)
# -------------------------------
FROM alpine:latest

# Install dependensi minimal
RUN apk add --no-cache bash ca-certificates postgresql-client

# Set working directory runtime
WORKDIR /app

# Copy binary hasil build dari stage builder
COPY --from=builder /bin/app ./

# Copy CLI soda agar bisa migrasi
COPY --from=builder /go/bin/soda /usr/local/bin/soda

# Copy file konfigurasi database dan folder migrasi
COPY --from=builder /src/WEB_UAKI/database.yml ./database.yml
COPY --from=builder /src/WEB_UAKI/migrations ./migrations

# Set environment agar bisa diakses dari luar
ENV ADDR=0.0.0.0
EXPOSE 3000

# Jalankan migrasi dulu, baru jalankan aplikasi Buffalo
CMD ["bash", "-c", "cd /app && echo '📦 Running migrations...' && soda migrate up && echo '🚀 Starting Buffalo app...' && ./app"]
