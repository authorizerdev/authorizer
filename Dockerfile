FROM golang:1.19.5-alpine as go-builder
WORKDIR /authorizer
COPY server server
COPY Makefile .

ARG VERSION="latest"
ENV VERSION="$VERSION"

RUN echo "$VERSION"
RUN apk add build-base &&\
    make clean && make && \
    chmod 777 build/server

FROM node:17-alpine3.12 as node-builder
WORKDIR /authorizer
COPY app app
COPY dashboard dashboard
COPY Makefile .
RUN apk add build-base &&\
    make build-app && \
    make build-dashboard

FROM alpine:latest
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
