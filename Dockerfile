FROM node:24-slim AS node-source
FROM oven/bun:1 AS bun-source

# start build stage with official golang image
FROM golang:1.26-trixie AS builder
WORKDIR /app

# retrieve dependencies + checksum
COPY go.* ./
RUN go mod download

# copy and build code
COPY . .
RUN CGO_ENABLED=0 go build -mod=readonly -v -o server .

# start runtime stage with Ubuntu Noble
# adds ca-certificates for HTTPS; copies node and bun from official images
FROM ubuntu:noble

RUN set -x && apt-get update && apt-get install -y \
    --no-install-recommends \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=node-source /usr/local/bin/node /usr/local/bin/node
COPY --from=bun-source  /usr/local/bin/bun  /usr/local/bin/bun

WORKDIR /app

# copy image to production and run binary
COPY --from=builder /app/server ./server
CMD ["./server"]
