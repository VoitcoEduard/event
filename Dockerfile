# syntax=docker/dockerfile:1
# ────────────────────────────────────────────────────────────────────
# Stage 1 – Build the Vue frontend
# ────────────────────────────────────────────────────────────────────
FROM --platform=$BUILDPLATFORM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

COPY frontend/package*.json ./
RUN --mount=type=cache,target=/root/.npm npm ci

COPY frontend/ ./

# VITE_API_URL is empty → the built app calls the same host/port as it's
# served from, so the Go server on :8080 handles both API and static files.
RUN VITE_API_URL="" npm run build
# Output: /app/frontend/dist


# ────────────────────────────────────────────────────────────────────
# Stage 2 – Build the Go backend (AMD64 for Asustor NAS)
# ────────────────────────────────────────────────────────────────────
FROM --platform=$BUILDPLATFORM golang:alpine AS backend-builder

WORKDIR /app/backend

ENV GOCACHE=/root/.cache/go-build
ENV GOMODCACHE=/root/.cache/go-mod

ARG DEBUG_MODE=false

COPY backend/go.mod backend/go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-mod go mod download

COPY backend/ ./

RUN --mount=type=cache,target=/root/.cache/go-mod \
    --mount=type=cache,target=/root/.cache/go-build \
    if [ "$DEBUG_MODE" = "true" ]; then \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags debug -o server .; \
    else \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o server .; \
    fi


# ────────────────────────────────────────────────────────────────────
# Stage 3 – Minimal final image
# ────────────────────────────────────────────────────────────────────
FROM --platform=linux/amd64 alpine:latest

# Install libwebp-tools (cwebp) for WebP image processing
RUN apk add --no-cache libwebp-tools ca-certificates

WORKDIR /root/

# Copy Go binary
COPY --from=backend-builder /app/backend/server ./server

# Copy built Vue app into ./dist  (Go serves this on "/")
COPY --from=frontend-builder /app/frontend/dist ./dist

EXPOSE 8080

CMD ["./server"]
