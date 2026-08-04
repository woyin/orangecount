# Development-only Fava 1.30.12 visual authority. This image is never part of
# the OrangeCount runtime or release artifacts.
FROM node:22.18.0-bookworm-slim AS node
FROM python:3.12.8-slim-bookworm

ARG NODE_VERSION=22.18.0
ARG FAVA_COMMIT=aa7538e8971252c9efc52c8a516a3a77d604553f

ENV LANG=en_US.UTF-8 \
    LANGUAGE=en_US:en \
    LC_ALL=en_US.UTF-8 \
    TZ=UTC \
    HOME=/tmp \
    PYTHONDONTWRITEBYTECODE=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    PLAYWRIGHT_BROWSERS_PATH=/ms-playwright

RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      bison build-essential ca-certificates chromium curl flex git locales pkg-config tzdata \
      fonts-noto-cjk \
 && sed -i 's/^# *en_US.UTF-8 UTF-8/en_US.UTF-8 UTF-8/' /etc/locale.gen \
 && locale-gen \
 && ln -snf /usr/share/zoneinfo/UTC /etc/localtime \
 && echo UTC > /etc/timezone \
 && rm -rf /var/lib/apt/lists/*

COPY --from=node /usr/local/bin/node /usr/local/bin/node
COPY --from=node /usr/local/lib/node_modules /usr/local/lib/node_modules
RUN ln -s ../lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm \
 && ln -s ../lib/node_modules/npm/bin/npx-cli.js /usr/local/bin/npx \
 && node --version \
 && npm --version

WORKDIR /app
RUN git clone --filter=blob:none https://github.com/beancount/fava.git /opt/fava \
 && git -C /opt/fava checkout --detach "${FAVA_COMMIT}" \
 && test "$(git -C /opt/fava rev-parse HEAD)" = "${FAVA_COMMIT}" \
 && pip install --no-cache-dir /opt/fava

COPY web/package.json web/package-lock.json /app/web/
RUN cd /app/web && npm ci --ignore-scripts

COPY tools/fava-reference/entrypoint.sh tools/fava-reference/write-environment-lock.mjs /app/
COPY web/playwright.config.mjs /app/web/
COPY web/tests /app/web/tests
RUN chmod 0555 /app/entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh"]
