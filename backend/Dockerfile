FROM --platform=linux/amd64 golang:alpine AS builder

WORKDIR /app

ENV GOCACHE=/root/.cache/go-build
ENV GOMODCACHE=/root/.cache/go-mod

# Build argument for debug mode (default: false for production)
ARG DEBUG_MODE=false

COPY go.mod ./

RUN go mod download

COPY . .

# Build with or without debug tags based on DEBUG_MODE argument
RUN if [ "$DEBUG_MODE" = "true" ]; then \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags debug -o main .; \
    else \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main .; \
    fi

FROM --platform=linux/amd64 alpine:latest

WORKDIR /root/

COPY --from=builder /app/main .

EXPOSE 8080

CMD ["./main"]
