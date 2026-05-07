# syntax=docker/dockerfile:1.7

# Build the NSQLite binary with CGO enabled because the project embeds SQLite C sources.
FROM golang:1.26-trixie AS builder

ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=1970-01-01T00:00:00Z

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Fetch the pinned SQLite amalgamation and produce stripped release binaries.
RUN mkdir -p /out && \
  go run ./scripts/fetch-sqlite/main.go && \
  CGO_ENABLED=1 go build \
    -trimpath \
    -ldflags="-s -w -X github.com/varavelio/nsqlite/internal/version.Version=${VERSION} -X github.com/varavelio/nsqlite/internal/version.Commit=${COMMIT} -X github.com/varavelio/nsqlite/internal/version.Date=${DATE}" \
    -o /out/nsqlite \
    ./cmd/nsqlite && \
  CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/entrypoint \
    ./cmd/entrypoint

# Download Litestream separately so the final image only receives the verified binary.
FROM debian:trixie-slim AS litestream

ARG DEBIAN_FRONTEND=noninteractive
ARG LITESTREAM_VERSION=0.5.11

RUN apt-get update && \
  apt-get install -y --no-install-recommends ca-certificates curl && \
  update-ca-certificates && \
  rm -rf /var/lib/apt/lists/*

WORKDIR /tmp/litestream

# Resolve the Debian architecture, verify the published checksum, and extract the binary.
RUN mkdir -p /out && \
  case "$(dpkg --print-architecture)" in \
    amd64) litestream_arch="x86_64" ;; \
    arm64) litestream_arch="arm64" ;; \
    *) echo "unsupported Debian architecture: $(dpkg --print-architecture)" >&2; exit 1 ;; \
  esac && \
  asset="litestream-${LITESTREAM_VERSION}-linux-${litestream_arch}.tar.gz" && \
  curl -fsSLO "https://github.com/benbjohnson/litestream/releases/download/v${LITESTREAM_VERSION}/${asset}" && \
  curl -fsSLO "https://github.com/benbjohnson/litestream/releases/download/v${LITESTREAM_VERSION}/checksums.txt" && \
  grep " ${asset}$" checksums.txt | sha256sum -c - && \
  tar -xzf "${asset}" && \
  install -m 0755 litestream /out/litestream

# Final runtime image: Debian Trixie slim, UTC-only clock, working CA store, and the binaries.
FROM debian:trixie-slim

ARG DEBIAN_FRONTEND=noninteractive
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=1970-01-01T00:00:00Z

LABEL org.opencontainers.image.title="NSQLite" \
  org.opencontainers.image.authors="Varavel" \
  org.opencontainers.image.description="SQLite over the network with optional Litestream integration" \
  org.opencontainers.image.url="https://github.com/varavelio/nsqlite" \
  org.opencontainers.image.documentation="https://github.com/varavelio/nsqlite" \
  org.opencontainers.image.source="https://github.com/varavelio/nsqlite" \
  org.opencontainers.image.licenses="MIT" \
  org.opencontainers.image.vendor="Varavel" \
  org.opencontainers.image.version="${VERSION}" \
  org.opencontainers.image.revision="${COMMIT}" \
  org.opencontainers.image.created="${DATE}"

# Keep the runtime defaults container-friendly while allowing users to override them.
ENV TZ=Etc/UTC \
  NSQLITE_DATA_DIR=/data \
  NSQLITE_LISTEN_HOST=0.0.0.0 \
  NSQLITE_LITESTREAM_ENABLED=false

# Install only the runtime dependencies we need and pin the timezone strictly to UTC.
RUN apt-get update && \
  apt-get install -y --no-install-recommends ca-certificates tzdata && \
  ln -snf /usr/share/zoneinfo/Etc/UTC /etc/localtime && \
  printf 'Etc/UTC\n' > /etc/timezone && \
  update-ca-certificates && \
  rm -rf /var/lib/apt/lists/*

# Prepare the default data directory expected by NSQLite.
RUN mkdir -p /data

COPY --from=builder /out/entrypoint /usr/local/bin/entrypoint
COPY --from=builder /out/nsqlite /usr/local/bin/nsqlite
COPY --from=litestream /out/litestream /usr/local/bin/litestream

# The entrypoint decides whether to run NSQLite directly or wrap it with Litestream.
RUN chmod 0755 /usr/local/bin/entrypoint

WORKDIR /data

EXPOSE 9876

ENTRYPOINT ["/usr/local/bin/entrypoint"]
