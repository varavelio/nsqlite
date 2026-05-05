FROM alpine:3.23.3 AS fetcher

WORKDIR /fetcher

RUN \
  apk add --quiet curl unzip && \
  # Fetch dprint
  curl -fsSL https://dprint.dev/install.sh | sh && \
  mv /root/.dprint/bin/dprint ./dprint && chmod +x ./dprint && \
  # Fetch task
  sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /fetcher v3.50.0 && \
  chmod +x ./task && \
  # Fetch gofumpt
  wget --no-verbose -O gofumpt https://github.com/mvdan/gofumpt/releases/download/v0.9.2/gofumpt_v0.9.2_linux_amd64 && \
  chmod +x ./gofumpt

# Use golang as the base image
FROM golang:1.26-trixie

# Install Node.js
COPY --from=node:24.15.0-trixie /usr/local/ /usr/local/

# Install golangci-lint
COPY --from=golangci/golangci-lint:v2.12.1 /usr/bin/golangci-lint /usr/local/bin/golangci-lint

# Install tools from the fetcher stage
COPY --from=fetcher /fetcher/dprint /usr/local/bin/dprint
COPY --from=fetcher /fetcher/task /usr/local/bin/task
COPY --from=fetcher /fetcher/gofumpt /usr/local/bin/gofumpt

# Set environment variables
ENV DEBIAN_FRONTEND="noninteractive"

RUN set -e && \
  # Update and install system dependencies
  apt-get update -qq && \
  apt-get install -yqq --no-install-recommends \
  ca-certificates wget curl zip unzip p7zip-full tzdata git tree ripgrep jq && \
  # Final setup: git config, and cleanup
  git config --global --add safe.directory '*' && \
  rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Add the startup script on every bash session
RUN rm -f /root/.bashrc
COPY .devcontainer/.bashrc /root/.bashrc

# Command just to keep the container running
CMD ["sleep", "infinity"]
