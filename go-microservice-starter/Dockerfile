# Stage 1: Build Go binary
FROM golang:1.23-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
# Allow Go toolchain to auto-upgrade if go.mod specifies a newer Go version
ENV GOTOOLCHAIN=auto
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/api ./cmd/api

# Stage 2: Minimal production runtime
FROM alpine:3.20
WORKDIR /app
RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/api ./api
COPY --from=builder /app/config ./config

EXPOSE 8080
CMD ["./api"]
