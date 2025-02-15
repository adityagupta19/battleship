# ---- Stage 1: Build the Go binary ----
    FROM golang:alpine AS builder

    WORKDIR /build
    
    COPY go.mod go.sum ./
    RUN go mod download
    
    COPY . .
    RUN go build -ldflags="-w -s" -o main ./cmd/
    
    # ---- Stage 2: Runtime stage ----
    FROM alpine:latest
    
    WORKDIR /app
    
    # Create user first
    RUN addgroup -S appgroup && adduser -S appuser -G appgroup
    
    # Copy binary from builder
    COPY --from=builder /build/main .
    
    # Set ownership and permissions
    RUN chown -R appuser:appgroup /app && \
        chmod +x /app/main
    
    USER appuser
    
    EXPOSE 50051
    
    CMD ["./main"]