# syntax=docker/dockerfile:1.7

# ------- Build stage ------------------------------------------------------
FROM golang:1.23-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod ./
# go.sum будет создан на первом tidy, если его ещё нет
COPY go.su[m] ./
RUN go mod download

COPY . .
# Догоняем зависимости на случай, если go.sum был пустой/устарел.
RUN go mod tidy

# Установить swag CLI и сгенерировать spec. Это перезапишет stub
# backend/internal/generated/docs/docs.go полноценной Swagger-спецификацией.
RUN go install github.com/swaggo/swag/cmd/swag@latest && \
    swag init -g cmd/main.go --output internal/generated/docs --parseInternal --parseDependency

# Сборка без debug-инфо, статическая
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/yulik3d ./cmd

# ------- Runtime stage ----------------------------------------------------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata curl \
    && addgroup -S yulik3d \
    && adduser -S -G yulik3d yulik3d

WORKDIR /app

COPY --from=builder /out/yulik3d /app/yulik3d
# Миграции встроены в бинарник через //go:embed, отдельно копировать не нужно.

USER yulik3d

EXPOSE 8080

ENTRYPOINT ["/app/yulik3d"]
