FROM golang:1.25-bookworm AS builder

WORKDIR /app

COPY module ./module

WORKDIR /app/module/core

RUN go mod download
RUN go build -trimpath -ldflags="-s -w" -o /out/mannaiah-api ./cmd/api

FROM debian:bookworm-slim AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /out/mannaiah-api /usr/local/bin/mannaiah-api

ENV CORE_HOST=0.0.0.0
ENV CORE_PORT=8080

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/mannaiah-api"]
