# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy dependency specifications
COPY go.mod go.sum ./
COPY vendor/ vendor/

# Copy application source code
COPY main.go main.go
COPY internal/ internal/

# Build static executable
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/dumbprotocol main.go

# Production stage
FROM alpine:3.19

WORKDIR /app

# Install runtime security certs & tzdata
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/dumbprotocol /app/dumbprotocol

# Default environment variables
ENV HOST=0.0.0.0
ENV PORT=8080
ENV APP_ENV=production

EXPOSE 8080

ENTRYPOINT ["/app/dumbprotocol"]
