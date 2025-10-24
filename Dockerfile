# Stage 1: Build
FROM golang:1.23.0 AS builder

ENV GOPROXY=https://proxy.golang.org,direct

WORKDIR /src/WEB_UAKI

# Copy dependency
COPY go.mod go.sum ./
RUN go mod download

# Install Buffalo CLI versi terbaru
RUN go install github.com/gobuffalo/cli/cmd/buffalo@latest

# Copy source code
COPY . .

# Build Buffalo app
RUN /go/bin/buffalo build --static -o /bin/app

# Stage 2: Runtime
FROM alpine:latest

RUN apk add --no-cache bash ca-certificates

WORKDIR /bin

# Copy hasil build
COPY --from=builder /bin/app .

# ✅ Copy file konfigurasi database
COPY --from=builder /src/WEB_UAKI/config ./config

ENV ADDR=0.0.0.0
EXPOSE 3000

CMD ["./app"]
