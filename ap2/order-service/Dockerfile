FROM golang:1.26-alpine AS builder
WORKDIR /app

# 1. Сначала копируем go.mod/go.sum сервиса И модуль proto
COPY order-service/go.mod order-service/go.sum ./order-service/
COPY proto/ ./proto/

# 2. Переходим в папку сервиса для скачивания зависимостей
WORKDIR /app/order-service
RUN go mod download

# 3. Копируем остальной исходный код сервиса
COPY order-service/ ./

# 4. Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -o order-service ./cmd/main

# ... (нижняя часть с FROM alpine и COPY --from=builder остается прежней)
FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/order-service .
EXPOSE 8080 9090
CMD ["./order-service"]
