# -------------------------------
# Stage 1: Build
# -------------------------------
FROM golang:1.23.0 AS builder

# Gunakan Go proxy
ENV GOPROXY=https://proxy.golang.org,direct

# Set working directory di dalam container
WORKDIR /src/WEB_UAKI

# Copy file dependensi terlebih dahulu
COPY go.mod go.sum ./
RUN go mod download

# Install Buffalo CLI versi terbaru dan pastikan binary tersedia di /usr/local/bin
RUN go install github.com/gobuffalo/cli/cmd/buffalo@latest \
    && cp $(go env GOPATH)/bin/buffalo /usr/local/bin/buffalo

# Copy seluruh source code project
COPY . .
COPY database.yml ./database.yml

# Jalankan proses build Buffalo
RUN /usr/local/bin/buffalo build --static -o /bin/app


# -------------------------------
# Stage 2: Runtime
# -------------------------------
FROM alpine:latest

# Install dependensi minimal
RUN apk add --no-cache bash ca-certificates

# Set working directory di runtime
WORKDIR /app

# Copy binary hasil build dari stage builder
COPY --from=builder /bin/app ./

# Copy Buffalo CLI dari builder agar bisa menjalankan migrasi
COPY --from=builder /usr/local/bin/buffalo /usr/local/bin/buffalo

# Copy file konfigurasi database
COPY --from=builder /src/WEB_UAKI/database.yml ./database.yml

# Set environment agar bisa diakses dari luar
ENV ADDR=0.0.0.0
EXPOSE 3000

# Jalankan aplikasi Buffalo dulu, lalu migrasi setelah 5 detik
CMD ["bash", "-c", "./app & sleep 5 && buffalo pop migrate up || true && wait"]
