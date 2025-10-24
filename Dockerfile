# Stage 1: Build
FROM golang:1.23.0 AS builder

ENV GOPROXY=https://proxy.golang.org,direct

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

RUN go install github.com/gobuffalo/cli/cmd/buffalo@latest

COPY . .

RUN /go/bin/buffalo build --static -o /bin/app

# Stage 2: Runtime
FROM alpine:latest

RUN apk add --no-cache bash ca-certificates

WORKDIR /app

COPY --from=builder /bin/app .
COPY --from=builder /app/config ./config

ENV ADDR=0.0.0.0
EXPOSE 3000

CMD ["./app"]
