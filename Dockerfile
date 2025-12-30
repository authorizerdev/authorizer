FROM docker.io/golang:1.24.2-alpine3.21 AS go-builder

WORKDIR /authorizer
COPY server server
COPY Makefile .

ARG VERSION="latest"
ENV VERSION="$VERSION"

RUN echo "$VERSION"
RUN apk add build-base &&\
    make clean && make && \
    chmod 777 build/server

FROM node:alpine AS node-builder
WORKDIR /authorizer
COPY app app
COPY dashboard dashboard
COPY Makefile .
RUN apk add build-base &&\
    make build-app && \
    make build-dashboard

FROM alpine:3.21
RUN adduser -D -h /authorizer -u 1000 -k /dev/null authorizer
WORKDIR /authorizer
RUN mkdir app dashboard
COPY --from=node-builder --chown=nobody:nobody /authorizer/app/build app/build
COPY --from=node-builder --chown=nobody:nobody /authorizer/app/favicon_io app/favicon_io
COPY --from=node-builder --chown=nobody:nobody /authorizer/dashboard/build dashboard/build
COPY --from=node-builder --chown=nobody:nobody /authorizer/dashboard/favicon_io dashboard/favicon_io
COPY --from=go-builder --chown=nobody:nobody /authorizer/build build
COPY templates templates
EXPOSE 8080
USER authorizer
CMD [ "./build/server" ]
