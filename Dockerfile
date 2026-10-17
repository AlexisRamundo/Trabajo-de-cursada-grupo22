# Etapa 1: compilacion
FROM golang:alpine AS builder

WORKDIR /app

ENV GOTOOLCHAIN=auto

# Descarga de dependencias optimizada en caché de capas
COPY go.mod go.sum ./
RUN go mod download

# Copiar el código fuente y compilar binario estático sin CGO
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/api main.go

# Etapa 2: imagen final liviana (solo el binario)
FROM alpine:3.19

# Crear usuario sin privilegios de root (práctica de seguridad)
RUN adduser -D -u 1000 app
USER app

COPY --from=builder /app/api /usr/local/bin/api

EXPOSE 8080

CMD ["api"]
