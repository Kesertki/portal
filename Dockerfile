# Step 1: Use the official Golang image as the builder image
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

COPY . .
# Build the application with CGO enabled (required for sqlite3)
RUN CGO_ENABLED=1 GOOS=linux go build -o main .

# Step 2: Create a smaller Docker image
FROM alpine:latest

WORKDIR /root/

# Create a directory for the database and set permissions
RUN mkdir -p /root/data && chmod 700 /root/data

COPY --from=builder /app/main .

# Set environment variable for the database path
ENV DATA_PATH=/root/data

# Optional: Add a health check
# HEALTHCHECK CMD curl --fail http://localhost:1323/health || exit 1

CMD ["./main"]
