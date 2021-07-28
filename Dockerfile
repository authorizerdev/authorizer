FROM golang:1.16-alpine as builder
WORKDIR /app
COPY . .

ARG VERSION=0.1.0-beta.0
ENV VERSION="${VERSION}"

RUN apk add build-base &&\
    cd server && \
    go mod download && \
    go build && \
    chmod 777 server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server/server .
EXPOSE 8080
CMD [ "./server" ]
