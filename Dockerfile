# start build stage with official golang image
FROM golang:1.26-trixie AS builder
WORKDIR /app

# retrieve dependencies + checksum
COPY go.* ./
RUN go mod download

# copy and build code
COPY . .
RUN CGO_ENABLED=0 go build -mod=readonly -v -o server .

# start runtime stage with Debian slim image
# adds ca-certificates for HTTPS; ignores and removes build bloat

FROM ubuntu:noble
RUN set -x && apt-get update && apt-get install -y \
    --no-install-recommends \
    ca-certificates curl unzip && \ 
    rm -rf /var/lib/apt/lists/*

# install latest node with fnm
RUN curl -fsSL https://fnm.vercel.app/install | bash -s -- --install-dir "/root/.fnm"

ENV FNM_PATH="/root/.fnm"
ENV PATH="$FNM_PATH:$PATH"

RUN eval "$(fnm env)" && fnm install v24 && fnm default v24

ENV PATH="/root/.fnm/aliases/default/bin:$PATH"

# install bun (for faster package management)
RUN curl -fsSL https://bun.sh/install | bash

ENV PATH="/root/.bun/bin:$PATH"

WORKDIR /app

# copy image to production and run binary
COPY --from=builder /app/server ./server
CMD ["./server"]
