
# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o app ./cmd/api

# Production stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/app .

EXPOSE 10000

CMD ["./app"]
