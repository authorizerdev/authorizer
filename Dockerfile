FROM golang:1.16-alpine as builder

WORKDIR /app

COPY . .

RUN apk add build-base && cd server && go mod download && go build && chmod 777 server && ls -l

EXPOSE 8080

ENTRYPOINT [ "/app/server/server" ]
