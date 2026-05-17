FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o payment-service ./cmd/main

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/payment-service .
EXPOSE 8081 9091
CMD ["./payment-service"]
