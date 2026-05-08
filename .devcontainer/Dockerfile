FROM alpine:3.23.3 AS fetcher

WORKDIR /fetcher

RUN \
  apk add --quiet curl unzip && \
  # Fetch task
  sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /fetcher v3.50.0 && \
  chmod +x ./task && \
  # Fetch hurl
  curl -fsSL https://github.com/Orange-OpenSource/hurl/releases/download/8.0.1/hurl-8.0.1-x86_64-unknown-linux-gnu.tar.gz | tar xz && \
  mv hurl-8.0.1-x86_64-unknown-linux-gnu/bin/hurl ./hurl && \
  # Fetch litestream
  curl -fsSL https://github.com/benbjohnson/litestream/releases/download/v0.5.11/litestream-0.5.11-linux-x86_64.tar.gz | tar xz && \
  mv litestream ./litestream && chmod +x ./litestream

# Use golang as the base image
FROM golang:1.26-trixie

# Install Node.js
COPY --from=node:24.15.0-trixie /usr/local/ /usr/local/

# Install golangci-lint
COPY --from=golangci/golangci-lint:v2.12.1 /usr/bin/golangci-lint /usr/local/bin/golangci-lint

# Install tools from the fetcher stage
COPY --from=fetcher /fetcher/task /usr/local/bin/task
COPY --from=fetcher /fetcher/hurl /usr/local/bin/hurl
COPY --from=fetcher /fetcher/litestream /usr/local/bin/litestream

# Set environment variables
ENV DEBIAN_FRONTEND="noninteractive"

RUN set -e && \
  # Update and install system dependencies
  apt-get update -qq && \
  apt-get install -yqq --no-install-recommends \
  ca-certificates wget curl zip unzip p7zip-full tzdata git tree ripgrep jq libxml2 \
  python3 python3-pip && \
  # Install mkdocs-material
  pip3 install --break-system-packages -qq mkdocs-material mkdocs-llmstxt && \
  # Final setup: git config, and cleanup
  git config --global --add safe.directory '*' && \
  rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Add the startup script on every bash session
RUN rm -f /root/.bashrc
COPY .devcontainer/.bashrc /root/.bashrc

# Command just to keep the container running
CMD ["sleep", "infinity"]
