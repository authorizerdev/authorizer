FROM golang:1.17-alpine as go-builder
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
COPY Makefile .
RUN apk add build-base &&\
    make build-app

FROM alpine:latest
WORKDIR /root/
RUN mkdir app
COPY --from=node-builder /authorizer/app/build app/build
COPY --from=go-builder /authorizer/build build
COPY templates templates
EXPOSE 8080
CMD [ "./build/server" ]
