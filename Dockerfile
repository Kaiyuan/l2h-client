# Build stage
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o l2h-cli main.go

# Runtime stage
FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /app/l2h-cli .
ENTRYPOINT ["./l2h-cli"]
