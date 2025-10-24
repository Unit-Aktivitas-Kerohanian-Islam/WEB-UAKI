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

# Install Buffalo CLI versi terbaru
RUN go install github.com/gobuffalo/cli/cmd/buffalo@latest

# Copy seluruh source code project
COPY . .

# (Opsional) Pastikan database.yml ikut di-copy
# Kalau file-nya ada di folder config/, ganti jadi COPY config/database.yml ./config/
COPY database.yml ./database.yml

# Jalankan proses build
RUN /go/bin/buffalo build --static -o /bin/app


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

# Copy file konfigurasi database dari stage builder
# Perhatikan: path-nya harus sama dengan di stage build
COPY --from=builder /src/WEB_UAKI/database.yml ./database.yml
# Jika database.yml kamu ada di dalam folder config/, gunakan:
# COPY --from=builder /src/WEB_UAKI/config ./config

# Set environment agar bisa diakses dari luar
ENV ADDR=0.0.0.0
EXPOSE 3000

# Jalankan aplikasi Buffalo
CMD ["./app"]
