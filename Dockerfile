FROM node:24-slim AS node-source

# start build stage with official golang image
FROM golang:1.26-trixie AS builder
WORKDIR /app

# retrieve dependencies + checksum
COPY go.* ./
RUN go mod download

# copy and build code
COPY . .
RUN go build -mod=readonly -v -o professor .

# start runtime stage
# adds ca-certificates for HTTPS; git for cloning repos;
# copies node and npm/npx from the official image, then installs pnpm via npm;
# pre-installs Playwright chromium-headless-shell and its OS dependencies
FROM debian:trixie-slim

RUN set -x && apt-get update && apt-get install -y \
    --no-install-recommends \
    ca-certificates \
    git \
    && rm -rf /var/lib/apt/lists/*

COPY --from=node-source /usr/local/bin/node  /usr/local/bin/node
COPY --from=node-source /usr/local/lib/      /usr/local/lib/

# recreate npm/npx as symlinks so Node resolves require() paths relative
# to the real file in node_modules, not /usr/local/bin/
RUN ln -sf /usr/local/lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm \
    && ln -sf /usr/local/lib/node_modules/npm/bin/npx-cli.js /usr/local/bin/npx

# install pnpm as the JS package manager (runs on the copied Node, no large binary)
RUN npm install -g pnpm@11 \
    && npm cache clean --force

# pre-install Playwright chromium-headless-shell (headless-only, smaller than full
# Chromium) and all required OS dependencies. Grading always runs headless, and a
# chromium project without an explicit `channel` resolves to the headless shell
# (Playwright v1.49+). NOTE: grader-controlled test configs must NOT set
# `channel: 'chromium'`, since the full headed browser is intentionally not installed.
#
# Install the browsers into a shared, root-owned path (not root's HOME cache) so
# the non-root runtime user can locate them. PLAYWRIGHT_BROWSERS_PATH is forwarded
# to the test subprocess by util.SandboxedEnv (PLAYWRIGHT_ prefix allowlist).
ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright
RUN npx playwright install --with-deps --only-shell chromium \
    && chmod -R a+rX /ms-playwright \
    && npm cache clean --force

WORKDIR /app

# copy image to production and run binary
COPY --from=builder /app/professor ./professor

# Run as an unprivileged user. Untrusted assessment code (pnpm lifecycle scripts,
# vite build) executes inside this container, so it must not run as root: a
# non-root uid limits the blast radius of a container escape and prevents the
# untrusted step from overwriting the professor binary or other container files.
# Playwright/pnpm write only to the per-job temp dirs and $HOME, so give the user
# a real home directory it owns; /app (including the professor binary) stays
# root-owned and read-only to the non-root user. The browsers in /ms-playwright
# are world-readable (chmod above), so the non-root user can execute them.
RUN useradd --create-home --uid 10001 --shell /usr/sbin/nologin professor
ENV HOME=/home/professor
USER professor

CMD ["./professor", "serve"]
