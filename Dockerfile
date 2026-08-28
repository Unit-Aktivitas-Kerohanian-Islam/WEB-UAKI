FROM golang:1.23-alpine AS builder
RUN apk add --no-cache git bash build-base
ENV GOPROXY=https://proxy.golang.org,direct
WORKDIR /src/WEB_UAKI
COPY go.mod go.sum ./
RUN go mod download
RUN go install github.com/gobuffalo/cli/cmd/buffalo@latest && \
    go install github.com/gobuffalo/pop/v6/soda@v6.1.1
COPY . .
COPY database.yml ./database.yml
RUN /go/bin/buffalo build --static -o /bin/app

FROM alpine:latest
RUN apk add --no-cache bash ca-certificates postgresql-client
WORKDIR /app
COPY --from=builder /bin/app ./
COPY --from=builder /go/bin/soda /usr/local/bin/soda
COPY --from=builder /src/WEB_UAKI/database.yml ./database.yml
COPY --from=builder /src/WEB_UAKI/migrations ./migrations
COPY --from=builder /src/WEB_UAKI/public ./public
ENV ADDR=0.0.0.0
EXPOSE 3000
CMD ["bash", "-c", "cd /app && soda migrate up && ./app task db:seed && ./app"]
