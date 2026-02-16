FROM golang:alpine AS builder

WORKDIR /app

ENV GOCACHE=/root/.cache/go-build
ENV GOMODCACHE=/root/.cache/go-mod

# Build argument for debug mode (default: true for development)
ARG DEBUG_MODE=true

COPY go.mod ./
# Copy go.sum if it exists, otherwise this line is fine or we can skip it if no dependencies yet
# COPY go.sum ./ 

RUN go mod download

COPY . .

# Build with or without debug tags based on DEBUG_MODE argument
RUN if [ "$DEBUG_MODE" = "true" ]; then \
    CGO_ENABLED=0 GOOS=linux go build -tags debug -o main .; \
    else \
    CGO_ENABLED=0 GOOS=linux go build -o main .; \
    fi

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/main .

EXPOSE 8080

CMD ["./main"]
