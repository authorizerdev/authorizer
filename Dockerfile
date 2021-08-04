FROM golang:1.16-alpine as builder
WORKDIR /app
COPY server server
COPY Makefile .

ARG VERSION=0.1.0-beta.0
ENV VERSION="${VERSION}"

RUN apk add build-base &&\
    make clean && make && \
    chmod 777 build/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY app app
COPY templates templates
COPY --from=builder /app/build build
EXPOSE 8080
CMD [ "./build/server" ]
