FROM golang:alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -ldflags="-w -s" -o main ./cmd/

FROM alpine:latest

WORKDIR /app

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /build/main .

RUN chown -R appuser:appgroup /app && chmod +x /app/main

USER appuser

EXPOSE 50053

CMD ["./main"]
