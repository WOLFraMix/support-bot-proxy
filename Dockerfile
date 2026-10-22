# Dockerfile
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
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/support-bot-proxy .

CMD ["./support-bot-proxy"]