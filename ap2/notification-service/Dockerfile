FROM golang:1.26-alpine AS builder
WORKDIR /app

COPY payment-service/go.mod payment-service/go.sum ./payment-service/
COPY proto/ ./proto/

WORKDIR /app/payment-service
RUN go mod download

COPY payment-service/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o payment-service ./cmd/main
# ...
WORKDIR /app

# 1. Копируем go.mod и go.sum из папки notification-service
COPY notification-service/go.mod notification-service/go.sum ./notification-service/

# 2. Если сервису уведомлений тоже нужны прото-файлы, раскомментируйте строку ниже:
# COPY proto/ ./proto/

# 3. Переходим в рабочую папку сервиса для скачивания зависимостей
WORKDIR /app/notification-service
RUN go mod download

# 4. Копируем весь остальной код сервиса
COPY notification-service/ ./

# 5. Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -o notification-service ./cmd/main

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/notification-service .
CMD ["./notification-service"]
