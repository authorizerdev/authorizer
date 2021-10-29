FROM golang:1.16-alpine as builder
WORKDIR /app
COPY server server
COPY Makefile .

ARG VERSION="latest"
ENV VERSION="$VERSION"

RUN echo "$VERSION"
RUN apk add build-base nodejs &&\
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
