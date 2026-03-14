# start build stage with official golang image
FROM golang:1.26-trixie AS builder
WORKDIR /app

# retrieve dependencies + checksum
COPY go.* ./
RUN go mod download

# copy and build code
COPY . .
RUN go build -mod=readonly -v -o server .

# start runtime stage with Debian slim image
# adds ca-certificates for HTTPS; ignores and removes build bloat
FROM debian:trixie-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    --no-install-recommends \
    ca-certificates && \ 
    rm -rf /var/lib/apt/lists/*
WORKDIR /app

# copy image to production and run binary
COPY --from=builder /app/server ./server
CMD ["./server"]
