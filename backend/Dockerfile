FROM --platform=$BUILDPLATFORM golang:alpine AS builder

WORKDIR /app

ENV GOCACHE=/root/.cache/go-build
ENV GOMODCACHE=/root/.cache/go-mod

# Build argument for debug mode (default: false for production)
ARG DEBUG_MODE=false
ARG TARGETARCH

COPY go.mod go.sum ./

RUN go mod download

COPY . .

# Build with or without debug tags based on DEBUG_MODE argument
RUN if [ "$DEBUG_MODE" = "true" ]; then \
    CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -tags debug -o main .; \
    else \
    CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -o main .; \
    fi

FROM alpine:latest

RUN apk add --no-cache ca-certificates libwebp-tools

RUN addgroup -S -g 10001 app && adduser -S -u 10001 -G app app
RUN mkdir -p /app/data/media && chown -R app:app /app/data

WORKDIR /app

COPY --chown=app:app --from=builder /app/main .

USER app

EXPOSE 8080

CMD ["./main"]
