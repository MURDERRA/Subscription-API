FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY . .

# go mod tidy скачивает зависимости и генерирует go.sum
RUN go mod tidy

# Устанавливаем swag CLI и генерируем docs/docs.go
# --parseInternal нужен чтобы swag видел типы из internal/model
RUN go install github.com/swaggo/swag/cmd/swag@latest && \
    swag init -g main.go -o docs --parseInternal

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server .

# ──────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/server .

RUN mkdir -p /app/logs

EXPOSE 8000

CMD ["./server"]
