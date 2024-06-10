FROM golang:latest AS builder

WORKDIR /app

# Copie apenas os arquivos de dependências
COPY go.mod go.sum ./
RUN go mod download

# Copie o restante do código
COPY . .

# Compile a aplicação
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o app .

# Use uma imagem mínima como base final, `alpine` por exemplo
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/app /app/app
COPY --from=builder /app/.env /app/.env
