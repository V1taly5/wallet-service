# Стадия сборки
FROM golang:1.23-alpine AS builder

# Установка переменных среды
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Установка необходимых утилит
RUN apk update && apk add --no-cache git

# Установка рабочей директории
WORKDIR /app

# Копирование go.mod и go.sum файлов
COPY go.mod ./

# Загрузка зависимостей
RUN go mod download

RUN apk del git

# Копирование исходного кода
COPY . .

# Сборка приложения
RUN go build -o wallet-service ./cmd/api

# Стадия запуска
FROM alpine:latest

# Установка необходимых утилит
RUN apk --no-cache add ca-certificates tzdata

# Установка рабочей директории
WORKDIR /app

# Копирование скомпилированного бинарника из стадии сборки
COPY --from=builder /app/wallet-service .

# Экспонирование порта
EXPOSE 8080

# Запуск приложения
CMD ["./wallet-service"]