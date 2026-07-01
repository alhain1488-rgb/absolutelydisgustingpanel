# Dockerized Xray node for local integration testing of the config generator.
# It bundles Go (to run the tagged integration test) and a real xray-core
# binary (to validate generated configs with `xray -test`).
#
# Build & run from the repo root:
#   docker build -t xray-node -f test/xray-node/Dockerfile .
#   docker run --rm xray-node
#
# The default command runs the integration test, which assembles a config with
# VLESS Reality + VMess + Trojan and asserts that `xray -test` accepts it.

FROM golang:1.25-bookworm

ARG XRAY_VERSION=v1.8.23
ARG TARGETARCH=64

# Install the xray-core binary from the official release.
RUN apt-get update && apt-get install -y --no-install-recommends unzip curl ca-certificates \
    && curl -fsSL -o /tmp/xray.zip \
       "https://github.com/XTLS/Xray-core/releases/download/${XRAY_VERSION}/Xray-linux-${TARGETARCH}.zip" \
    && unzip -o /tmp/xray.zip xray -d /usr/local/bin \
    && chmod +x /usr/local/bin/xray \
    && rm -rf /tmp/xray.zip /var/lib/apt/lists/*

ENV XRAY_BIN=/usr/local/bin/xray
WORKDIR /src/backend

# Cache modules first.
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

# Validate the generated config against a real xray binary.
CMD ["go", "test", "-tags", "integration", "-run", "TestGeneratedConfig_AcceptedByXray", "-v", "./internal/xrayconfig/..."]
