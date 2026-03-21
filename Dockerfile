# syntax=docker/dockerfile:1.4
# Use BuildKit for cache mounts (faster CI: DOCKER_BUILDKIT=1)
FROM golang:1.25-alpine3.23 AS go-builder
WORKDIR /authorizer

ARG TARGETPLATFORM
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION="latest"

ENV CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    VERSION=$VERSION

RUN apk add --no-cache ca-certificates tzdata

# Dependency cache: only re-run when go.mod/go.sum change
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Source code: rebuild binary only when code changes
COPY main.go ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY gqlgen.yml ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    mkdir -p build/${GOOS}/${GOARCH} && \
    go build -trimpath -mod=readonly -tags netgo -ldflags "-w -s -X main.VERSION=$VERSION" -o build/${GOOS}/${GOARCH}/authorizer . && \
    chmod 755 build/${GOOS}/${GOARCH}/authorizer

FROM alpine:3.23.3 AS node-builder
WORKDIR /authorizer
COPY web/app/package*.json web/app/
COPY web/dashboard/package*.json web/dashboard/
RUN apk add --no-cache nodejs npm
# Cache npm package tarballs across builds (faster re-installs in CI)
RUN --mount=type=cache,target=/root/.npm \
    npm config set cache /root/.npm && \
    cd web/app && npm ci --prefer-offline --no-audit && \
    cd ../dashboard && npm ci --prefer-offline --no-audit
COPY web/app web/app
COPY web/dashboard web/dashboard
RUN cd web/app && npm run build && cd ../dashboard && npm run build

FROM alpine:3.23.3

ARG TARGETARCH=amd64

# CA certificates for TLS connections (OAuth, webhooks, etc.)
COPY --from=go-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Timezone data
COPY --from=go-builder /usr/share/zoneinfo /usr/share/zoneinfo

WORKDIR /authorizer
COPY --from=node-builder /authorizer/web/app/build web/app/build
COPY --from=node-builder /authorizer/web/app/favicon_io web/app/favicon_io
COPY --from=node-builder /authorizer/web/dashboard/build web/dashboard/build
COPY --from=node-builder /authorizer/web/dashboard/favicon_io web/dashboard/favicon_io
COPY --from=go-builder /authorizer/build/linux/${TARGETARCH}/authorizer ./authorizer
COPY web/templates web/templates

RUN addgroup -g 1000 authorizer && \
    adduser -D -u 1000 -G authorizer authorizer && \
    chown -R authorizer:authorizer /authorizer

USER authorizer

EXPOSE 8080 8081
ENTRYPOINT [ "./authorizer" ]
CMD []
