# Fetch external release binaries once so later stages can stay focused and small.
FROM alpine:3.23.3 AS fetcher

ARG TASK_VERSION=3.50.0
ARG LITESTREAM_VERSION=0.5.11

WORKDIR /fetcher

# Keep all downloaded tools in /fetcher and verify Litestream before copying it forward.
RUN apk add --quiet --no-cache ca-certificates curl tar && \
  update-ca-certificates && \
  case "$(apk --print-arch)" in \
    x86_64) litestream_arch="x86_64" ;; \
    aarch64) litestream_arch="arm64" ;; \
    *) echo "unsupported Alpine architecture: $(apk --print-arch)" >&2; exit 1 ;; \
  esac && \
  sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /fetcher "v${TASK_VERSION}" && \
  chmod 0755 /fetcher/task && \
  asset="litestream-${LITESTREAM_VERSION}-linux-${litestream_arch}.tar.gz" && \
  curl -fsSLO "https://github.com/benbjohnson/litestream/releases/download/v${LITESTREAM_VERSION}/${asset}" && \
  curl -fsSLO "https://github.com/benbjohnson/litestream/releases/download/v${LITESTREAM_VERSION}/checksums.txt" && \
  grep " ${asset}$" checksums.txt | sha256sum -c - && \
  tar -xzf "${asset}" && \
  install -m 0755 litestream /fetcher/litestream

# Build NSQLite with the repository Taskfile, which owns version metadata and flags.
FROM golang:1.26-trixie AS builder

COPY --from=fetcher /fetcher/task /usr/local/bin/task

WORKDIR /src

COPY go.mod go.sum Taskfile.yml ./
RUN go mod download

COPY . .

# Taskfile computes VERSION, COMMIT, and DATE, then builds both release binaries.
RUN task deps build build:entrypoint && \
  mkdir -p /out && \
  cp ./dist/nsqlite ./dist/entrypoint /out/

# Final runtime image: Debian Trixie slim with only CA data, UTC timezone, and binaries.
FROM debian:trixie-slim

ARG DEBIAN_FRONTEND=noninteractive

LABEL org.opencontainers.image.title="NSQLite" \
  org.opencontainers.image.authors="Varavel" \
  org.opencontainers.image.description="SQLite over the network with optional Litestream integration" \
  org.opencontainers.image.url="https://github.com/varavelio/nsqlite" \
  org.opencontainers.image.documentation="https://github.com/varavelio/nsqlite" \
  org.opencontainers.image.source="https://github.com/varavelio/nsqlite" \
  org.opencontainers.image.licenses="MIT" \
  org.opencontainers.image.vendor="Varavel"

# Keep runtime defaults container-friendly while allowing users to override them.
ENV TZ=Etc/UTC \
  NSQLITE_DATA_DIR=/data \
  NSQLITE_LISTEN_HOST=0.0.0.0 \
  NSQLITE_LITESTREAM_ENABLED=false

# Install only runtime dependencies and pin the timezone strictly to UTC.
RUN apt-get update && \
  apt-get install -y --no-install-recommends ca-certificates tzdata && \
  ln -snf /usr/share/zoneinfo/Etc/UTC /etc/localtime && \
  printf 'Etc/UTC\n' > /etc/timezone && \
  update-ca-certificates && \
  mkdir -p /data && \
  rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/entrypoint /usr/local/bin/entrypoint
COPY --from=builder /out/nsqlite /usr/local/bin/nsqlite
COPY --from=fetcher /fetcher/litestream /usr/local/bin/litestream

WORKDIR /data

EXPOSE 9876

ENTRYPOINT ["/usr/local/bin/entrypoint"]
