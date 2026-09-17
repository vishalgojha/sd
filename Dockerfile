# syntax=docker/dockerfile:1
FROM golang:1.26-bookworm AS build

WORKDIR /src

RUN apt-get update && apt-get install -y --no-install-recommends gcc libc6-dev ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -buildvcs=false -trimpath -ldflags="-s -w" -o /out/sdsheetal ./...

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates wget \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=build /out/sdsheetal ./sdsheetal

ENV PORT=8080
ENV SDSHEETAL_DATA=/data
EXPOSE 8080
VOLUME ["/data"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1

ENTRYPOINT ["/app/sdsheetal"]