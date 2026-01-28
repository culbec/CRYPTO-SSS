FROM node:lts-bookworm

LABEL maintainer="Alexandru 'culbec' Profir <profir.alexx@gmail.com>"

### Start building
WORKDIR /app/frontend

COPY src/frontend/package.json src/frontend/package-lock.json ./

RUN npm ci

# Copy the rest of the source
COPY src/frontend ./

RUN apt-get update && apt-get install -y --no-install-recommends sudo && \
    rm -rf /var/lib/apt/lists/* && \
    groupadd --system app && useradd --system --gid app --shell /bin/bash --create-home crypto && \
    echo "crypto ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/crypto && \
    chmod 0440 /etc/sudoers.d/crypto

WORKDIR /app

RUN mkdir -p /app/logs \
    && chmod -R 755 /app \
    && chown -R crypto:app /app

USER crypto

EXPOSE 4200

ENTRYPOINT ["/app/entrypoint.sh"]