# Build stage
FROM golang:1.24-alpine AS builder

# Install git (only in builder) in case go modules require it
RUN apk add --no-cache git

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary (statically linked)
RUN CGO_ENABLED=0 go build -o /teleparty ./cmd

# Final runtime stage (distroless)
FROM gcr.io/distroless/static

# Set working dir
WORKDIR /

# Copy binary from builder
COPY --from=builder /teleparty /teleparty

# Run the app
CMD ["/teleparty"]
