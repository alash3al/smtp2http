FROM golang:1.14.4-alpine

RUN apk update && apk add git
RUN go get github.com/stouch/smtp2http

ENTRYPOINT ["smtp2http"]

WORKDIR /root/