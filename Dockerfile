FROM golang:1.24-alpine AS builder

WORKDIR /app

# Bağımlılıkları kopyala ve indir
COPY go.mod ./
RUN go mod download

# Kaynak kodları kopyala
COPY . .

# Uygulamayı derle
RUN CGO_ENABLED=0 GOOS=linux go build -o /url-shortener ./cmd/server

FROM alpine:3.18

WORKDIR /app
COPY --from=builder /url-shortener .
COPY web /app/web

EXPOSE 8080 50051
CMD ["./url-shortener"] 