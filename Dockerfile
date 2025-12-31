FROM golang:1.25.5-alpine3.23 as go-builder
WORKDIR /authorizer

# Copy go mod files for dependency resolution
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY main.go ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY gqlgen.yml ./

ARG VERSION="latest"
ENV VERSION="$VERSION"

RUN echo "$VERSION"
# Build the server binary
RUN apk add build-base && \
    go build -ldflags "-w -X main.VERSION=$(VERSION)" -o build/server . && \
    chmod 777 build/server

FROM node:25-alpine3.22 as node-builder
WORKDIR /authorizer
# Copy package files first for better layer caching
COPY web/app/package*.json web/app/
COPY web/dashboard/package*.json web/dashboard/
# Install dependencies
RUN cd web/app && npm ci && \
    cd ../dashboard && npm ci
# Copy source files
COPY web/app web/app
COPY web/dashboard web/dashboard
# Build applications
RUN cd web/app && npm run build && \
    cd ../dashboard && npm run build

FROM alpine:3.23
RUN adduser -D -h /authorizer -u 1000 -k /dev/null authorizer
WORKDIR /authorizer
RUN mkdir app dashboard
COPY --from=node-builder --chown=nobody:nobody /authorizer/web/app/build app/build
COPY --from=node-builder --chown=nobody:nobody /authorizer/web/app/favicon_io app/favicon_io
COPY --from=node-builder --chown=nobody:nobody /authorizer/web/dashboard/build dashboard/build
COPY --from=node-builder --chown=nobody:nobody /authorizer/web/dashboard/favicon_io dashboard/favicon_io
COPY --from=go-builder --chown=nobody:nobody /authorizer/build build
COPY templates templates
EXPOSE 8080 8081
USER authorizer
# Use ENTRYPOINT to allow passing CLI arguments
ENTRYPOINT [ "./build/server" ]
CMD []
