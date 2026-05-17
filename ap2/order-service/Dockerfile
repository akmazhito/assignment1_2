FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o order-service ./cmd/main

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/order-service .
EXPOSE 8080 9090
CMD ["./order-service"]
