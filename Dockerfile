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

# start runtime stage
# adds ca-certificates for HTTPS; git for cloning repos;
# copies node, npm/npx, and bun from official images;
# pre-installs Playwright Chromium and its OS dependencies
FROM debian:trixie-slim

RUN set -x && apt-get update && apt-get install -y \
    --no-install-recommends \
    ca-certificates \
    git \
    && rm -rf /var/lib/apt/lists/*

COPY --from=node-source /usr/local/bin/node  /usr/local/bin/node
COPY --from=node-source /usr/local/lib/      /usr/local/lib/
COPY --from=bun-source  /usr/local/bin/bun   /usr/local/bin/bun

# recreate npm/npx as symlinks so Node resolves require() paths relative
# to the real file in node_modules, not /usr/local/bin/
RUN ln -sf /usr/local/lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm \
    && ln -sf /usr/local/lib/node_modules/npm/bin/npx-cli.js /usr/local/bin/npx

# pre-install Playwright Chromium and all required OS dependencies
RUN npx playwright install --with-deps chromium \
    && npm cache clean --force

WORKDIR /app

# copy image to production and run binary
COPY --from=builder /app/server ./server
CMD ["./server"]
