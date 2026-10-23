# ----------------------------
# Stage 1: Build the operator app's web UI
# ----------------------------
FROM node:24-alpine AS operator-ui

WORKDIR /ui
COPY cmd/operator/frontend/package.json cmd/operator/frontend/package-lock.json ./
# Cache mounts speed up local rebuilds; they never end up in the image.
RUN --mount=type=cache,target=/root/.npm npm ci
COPY cmd/operator/frontend/ ./
RUN npm run build

# ----------------------------
# Stage 2: Build the Go binaries
# ----------------------------
FROM golang:1.27-alpine AS builder

WORKDIR /app

# Copy Go dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the Go source code
COPY . .

# The operator binary embeds the web UI built above
COPY --from=operator-ui /ui/dist ./cmd/operator/frontend/dist

# Build the API server, the data import CLI and the operator app. The Go build
# cache is kept between builds, so only changed packages recompile. (Modules
# stay in the layer above, which CI's layer cache restores.)
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server \
 && CGO_ENABLED=0 GOOS=linux go build -o /out/migrate ./cmd/migrate \
 && CGO_ENABLED=0 GOOS=linux go build -o /out/operator ./cmd/operator

# ----------------------------
# Stage 3: Final Production Image
# ----------------------------
FROM alpine:latest

WORKDIR /app

# Certificates for external calls; poppler-utils provides pdftotext for PDF imports
RUN apk --no-cache add ca-certificates poppler-utils

# Set the default data directory and port inside the container
ENV DATA_DIR=/app/data \
    PORT=8080

COPY --from=builder /out/server /out/migrate /out/operator ./

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1

CMD ["./server"]
