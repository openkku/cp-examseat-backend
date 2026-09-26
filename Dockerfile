# ----------------------------
# Stage 1: Build the Go binaries
# ----------------------------
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy Go dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the Go source code
COPY . .

# Build the API server and the data import CLI
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server \
 && CGO_ENABLED=0 GOOS=linux go build -o /out/migrate ./cmd/migrate

# ----------------------------
# Stage 2: Final Production Image
# ----------------------------
FROM alpine:latest

WORKDIR /app

# Install certificates for external API calls
RUN apk --no-cache add ca-certificates

# Set the default data directory and port inside the container
ENV DATA_DIR=/app/data \
    PORT=8080

COPY --from=builder /out/server /out/migrate ./

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1

CMD ["./server"]
