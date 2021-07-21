FROM golang:1.16-alpine as builder

WORKDIR /app

COPY . .

RUN apk add build-base && cd server && go mod download && go build && chmod 777 server && ls -l

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server/server .
EXPOSE 8080
CMD [ "./server" ]
