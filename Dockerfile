# syntax=docker/dockerfile:1

# --- Build stage ---
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install git (for go mod) and build tools
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go app (adjust -o if your binary should be named differently)
RUN CGO_ENABLED=0 GOOS=linux go build -o teleparty ./cmd/main.go

# --- Run stage ---
FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder
COPY --from=builder /app/teleparty .

# Expose the port your Gin app listens on (adjust if needed)
EXPOSE 8080

# Run the binary
ENTRYPOINT ["./teleparty"]
