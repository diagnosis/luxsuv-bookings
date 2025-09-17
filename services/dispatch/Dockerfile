FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the service
RUN CGO_ENABLED=0 GOOS=linux go build -o dispatch ./services/dispatch

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

# Copy the binary
COPY --from=builder /app/dispatch .

# Expose port
EXPOSE 8083

CMD ["./dispatch"]