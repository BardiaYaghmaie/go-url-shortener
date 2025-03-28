# Build stage
FROM golang:1.23.5-alpine AS builder

WORKDIR /app

# Install required dependencies for SQLite
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o url-shortener .

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

COPY --from=builder /app/url-shortener .
COPY --from=builder /app/data ./data

EXPOSE 8080

CMD ["./url-shortener"]