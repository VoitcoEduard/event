# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

WORKDIR /src
ARG TARGETOS
ARG TARGETARCH

COPY go.mod ./
COPY main.go ./

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /deploy-control .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app app \
    && mkdir -p /data \
    && chown app:app /data

COPY --chown=app:app --from=builder /deploy-control /usr/local/bin/deploy-control

USER app
EXPOSE 8090
VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/deploy-control"]
