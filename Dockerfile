FROM golang:1.14.4 as builder
WORKDIR /go/src/build
COPY . .
ENV CGO_ENABLED=0
RUN GOOS=linux GOARCH=arm64 go build -mod vendor -a -o smtp2http .

FROM arm64v8/alpine  
WORKDIR /root/
COPY --from=builder /go/src/build/smtp2http .
ENV WEBHOOK=http://localhost:8080/api/smtp-hook
ENV LISTEN_PORT=25
ENV DOMAIN=localhost.com
CMD ./smtp2http --listen=:${LISTEN_PORT} --webhook=${WEBHOOK} --domain=${DOMAIN}