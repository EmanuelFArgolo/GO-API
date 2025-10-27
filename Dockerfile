# ----- Estágio 1: Build -----
# Usamos a imagem oficial do Go com Alpine (leve)
FROM golang:1.23.0-alpine AS builder

# Define o diretório de trabalho dentro do container
WORKDIR /app

# Copia os arquivos de módulo e baixa as dependências primeiro
# Isso otimiza o cache do Docker
COPY go.mod go.sum ./
RUN go mod download

# Copia todo o resto do código-fonte
COPY . .

# Compila a aplicação
# CGO_ENABLED=0 cria um binário estático (essencial para Alpine)
# -o /app/quiz-api define o nome do arquivo de saída
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/quiz-api ./cmd/server/main.go

# ----- Estágio 2: Final -----
# Usamos a imagem Alpine mais básica
FROM alpine:latest

# Adiciona certificados SSL/TLS (CRÍTICO para chamar APIs de LLM)
RUN apk --no-cache add ca-certificates

# Define o diretório de trabalho
WORKDIR /root/

# Copia APENAS o binário compilado do estágio 'builder'
COPY --from=builder /app/quiz-api .


COPY internal/store/migrations ./internal/store/migrations/
# Expõe a porta que a nossa API vai ouvir
EXPOSE 8080

# Comando para rodar a aplicação
CMD ["./quiz-api"]