FROM golang:1.14.4-alpine

RUN apk update && apk add git

# All this is quite ugly, should be cleaned up

RUN git clone https://github.com/stouch/smtp2http $GOPATH/src

WORKDIR $GOPATH/src

RUN go mod vendor
ENV CGO_ENABLED=0
RUN GOOS=linux GOARCH=arm64 go build -mod vendor -a -o smtp2http .
RUN mv $GOPATH/src/smtp2http /root/smtp2http

WORKDIR /root/

ENTRYPOINT ["/root/smtp2http"]
