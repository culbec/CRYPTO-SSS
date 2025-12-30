FROM golang:1.25-alpine AS builder

LABEL maintainer="Alexandru 'culbec' Profir <profir.alexx@gmail.com>"

# HuggingFace configs are loaded using provided
ARG HF_REPO="culbec/CRYPTO-SSS"
ENV HF_REPO=${HF_REPO}

ENV PYTHON_UNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    POETRY_VIRTUALENVS_CREATE=false \
    POETRY_CACHE_DIR=/tmp/poetry_cache

# Install git, python3, poetry and huggingface-hub[cli]
RUN apk update && apk add --no-cache git python3 py3-pip && \
    pip install --break-system-packages poetry huggingface_hub[cli]

# Bypass proxy.golang.org (workaround for HTTP/2 INTERNAL_ERROR from the proxy)
ENV GOPROXY=direct

### Start building
WORKDIR /src/backend

# Cache dependencies first
COPY src/backend/go.mod src/backend/go.sum ./
RUN go mod download

# Copy the rest of the source
COPY src/backend/ ./

# Downloading the assets from the HuggingFace repository
RUN --mount=type=secret,id=hf_token,target=/run/secrets/HF_TOKEN \
    HF_TOKEN="$(cat /run/secrets/HF_TOKEN)" && \
    python3 ./deployments/hf_assets.py --token "$HF_TOKEN" --repo-id "$HF_REPO"

# Build the backend binary (entrypoint package is ./cmd)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/backend ./cmd

FROM alpine:3.20 AS runtime

LABEL maintainer="Alexandru 'culbec' Profir <profir.alexx@gmail.com>"

RUN addgroup -S app && adduser -S backend -G app

WORKDIR /app

COPY --from=builder /out/backend /app/backend
COPY src/backend/configs /app/configs
COPY src/backend/deployments/entrypoint.sh /app/entrypoint.sh

# Default logging path is logs/backend.log; ensure directory exists and is writable.
# Ensure the entrypoint permissions are set correctly
RUN mkdir -p /app/logs \
    && chmod 755 /app /app/configs /app/logs /app/entrypoint.sh \
    && chown -R backend:app /app && chown backend:app /app/entrypoint.sh

USER backend

EXPOSE 3000

ENTRYPOINT ["/app/entrypoint.sh"]