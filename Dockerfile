FROM golang:1.14.4 as builder
RUN git clone https://github.com/alash3al/smtp2http /go/src/build
WORKDIR /go/src/build
RUN go mod vendor
ENV CGO_ENABLED=0
RUN GOOS=linux go build -mod vendor -a -o smtp2http .

FROM golang:1.16.2  
WORKDIR /root/
COPY --from=builder /go/src/build/smtp2http /usr/bin/smtp2http
ENTRYPOINT ["smtp2http"]
