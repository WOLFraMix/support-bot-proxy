FROM golang:1.27-alpine AS builder

WORKDIR /app

# Копируем go.mod/go.sum и скачиваем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь код проекта
COPY . .

# Собираем бинарь
RUN CGO_ENABLED=0 GOOS=linux go build -o support-bot-proxy ./cmd/server

# Финальный образ — маленький, только бинарь
FROM alpine:3.24

# Добавляем ca-certificates и curl (нужен для healthcheck)
RUN apk --no-cache add ca-certificates curl

WORKDIR /root/

# Копируем бинарь из builder
COPY --from=builder /app/support-bot-proxy .

# Важно: явно говорим, какой порт слушает приложение
EXPOSE 8080

# Делаем бинарь исполняемым (на всякий случай)
RUN chmod +x ./support-bot-proxy

CMD ["./support-bot-proxy"]