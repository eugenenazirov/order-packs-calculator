ARG GO_VERSION=1.25.1
ARG ALPINE_VERSION=3.20

FROM golang:${GO_VERSION}-alpine AS builder

RUN apk add --no-cache build-base git

WORKDIR /src

COPY go.mod ./
COPY internal ./internal
COPY cmd ./cmd
COPY web ./web

ENV CGO_ENABLED=0 \
    GO111MODULE=on

RUN go build -o /out/pack-calculator ./cmd/server

FROM alpine:${ALPINE_VERSION} AS runner

RUN addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=builder /out/pack-calculator /usr/local/bin/pack-calculator
COPY --from=builder /src/web ./web

ENV PORT=8080 \
    PACK_SIZES="250,500,1000,2000,5000"

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:${PORT}/api/health >/dev/null 2>&1 || exit 1

USER app

ENTRYPOINT ["/usr/local/bin/pack-calculator"]
