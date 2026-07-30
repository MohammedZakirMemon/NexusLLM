# Stage 1: Build
FROM golang:1.22-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /nexusllm ./cmd/gateway

# Stage 2: Runtime (minimal scratch image)
FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /nexusllm /nexusllm
EXPOSE 8080
ENTRYPOINT ["/nexusllm"]
