# syntax=docker/dockerfile:1.4
# Use BuildKit for cache mounts (faster CI: DOCKER_BUILDKIT=1)
FROM golang:1.24-alpine3.23 as go-builder
WORKDIR /authorizer

ARG TARGETPLATFORM
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION="latest"

ENV CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    VERSION=$VERSION

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
RUN apk add --no-cache build-base
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    mkdir -p build/${GOOS}/${GOARCH} && \
    go build -trimpath -mod=readonly -tags netgo -ldflags "-w -s -X main.VERSION=$VERSION" -o build/${GOOS}/${GOARCH}/authorizer . && \
    chmod 755 build/${GOOS}/${GOARCH}/authorizer

FROM alpine:3.23.3 as node-builder
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

RUN apk update && apk upgrade --no-cache && \
    adduser -D -h /home/authorizer -u 1000 -k /dev/null authorizer && \
    mkdir -p web/app web/dashboard
WORKDIR /authorizer
COPY --from=node-builder --chown=nobody:nobody /authorizer/web/app/build web/app/build
COPY --from=node-builder --chown=nobody:nobody /authorizer/web/app/favicon_io web/app/favicon_io
COPY --from=node-builder --chown=nobody:nobody /authorizer/web/dashboard/build web/dashboard/build
COPY --from=node-builder --chown=nobody:nobody /authorizer/web/dashboard/favicon_io web/dashboard/favicon_io
COPY --from=go-builder --chown=nobody:nobody /authorizer/build/linux/${TARGETARCH}/authorizer ./authorizer
COPY web/templates web/templates
EXPOSE 8080 8081
USER authorizer
# ENTRYPOINT allows docker run args to be passed to the authorizer binary.
# When extending this image with a shell-form CMD (e.g. to expand env vars for Railway),
# override ENTRYPOINT in your Dockerfile: ENTRYPOINT ["/bin/sh", "-c"] so CMD runs in a shell.
ENTRYPOINT [ "./authorizer" ]
CMD []
