FROM golang:1.25.7-alpine3.23 as go-builder
WORKDIR /authorizer

ARG TARGETPLATFORM
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION="latest"

ENV CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    VERSION=$VERSION

# Copy go mod files for dependency resolution
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY main.go ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY gqlgen.yml ./

RUN echo "$VERSION"
# Build the server binary (upgrade apk index for security)
RUN apk update && apk upgrade --no-cache && apk add --no-cache build-base && \
    mkdir -p build/${GOOS}/${GOARCH} && \
    go build -trimpath -mod=readonly -tags netgo -ldflags "-w -s -X main.VERSION=$VERSION" -o build/${GOOS}/${GOARCH}/authorizer . && \
    chmod 755 build/${GOOS}/${GOARCH}/authorizer

FROM alpine:3.23.3 as node-builder
WORKDIR /authorizer
# Copy package files first for better layer caching
COPY web/app/package*.json web/app/
COPY web/dashboard/package*.json web/dashboard/
# Install Node.js, npm, and dependencies with upgraded base for security
RUN apk update && apk upgrade --no-cache && \
    apk add --no-cache nodejs npm && \
    cd web/app && npm ci && \
    cd ../dashboard && npm ci
# Copy source files
COPY web/app web/app
COPY web/dashboard web/dashboard
# Build applications
RUN cd web/app && npm run build && \
    cd ../dashboard && npm run build

FROM alpine:3.23.3

ARG TARGETARCH=amd64

RUN apk update && apk upgrade --no-cache && \
    adduser -D -h /home/authorizer -u 1000 -k /dev/null authorizer
WORKDIR /authorizer
RUN mkdir -p web/app web/dashboard
COPY --from=node-builder --chown=nobody:nobody /authorizer/web/app/build web/app/build
COPY --from=node-builder --chown=nobody:nobody /authorizer/web/app/favicon_io web/app/favicon_io
COPY --from=node-builder --chown=nobody:nobody /authorizer/web/dashboard/build web/dashboard/build
COPY --from=node-builder --chown=nobody:nobody /authorizer/web/dashboard/favicon_io web/dashboard/favicon_io
COPY --from=go-builder --chown=nobody:nobody /authorizer/build/linux/${TARGETARCH}/authorizer ./authorizer
COPY web/templates web/templates
EXPOSE 8080 8081
USER authorizer
# Use ENTRYPOINT to allow passing CLI arguments
ENTRYPOINT [ "./authorizer" ]
CMD []
