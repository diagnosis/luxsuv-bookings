FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the service
RUN CGO_ENABLED=0 GOOS=linux go build -o payments ./services/payments

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

# Copy the binary
COPY --from=builder /app/payments .

# Expose port
EXPOSE 8085

CMD ["./payments"]