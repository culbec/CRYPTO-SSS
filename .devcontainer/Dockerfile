FROM golang:1.25-bookworm

ENV GOPATH=/home/crypto/go \
    GOMODCACHE=/home/crypto/go/pkg/mod \
    PATH="/home/crypto/go/bin:${PATH}"

RUN apt-get update && apt-get install -y --no-install-recommends \
    nodejs npm sudo \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system app && useradd --system --gid app --shell /bin/bash --create-home crypto \
    && echo "crypto ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/crypto \
    && chmod 0440 /etc/sudoers.d/crypto \
    && su crypto -c "go install github.com/air-verse/air@latest"

USER crypto
WORKDIR /workspaces/crypto-sss
