package main

import "flag"

var (
	flagServerName     = flag.String("name", "smtp2http", "the server name")
	flagListenAddr     = flag.String("listen", ":smtp", "the smtp address to listen on")
	flagWebhook        = flag.String("webhook", "http://localhost:8080/my/webhook", "the webhook to send the data to")
	flagMaxMessageSize = flag.Int64("msglimit", 1024*1024*2, "maximum incoming message size")
	flagReadTimeout    = flag.Int("timeout.read", 5, "the read timeout in seconds")
	flagWriteTimeout   = flag.Int("timeout.write", 5, "the write timeout in seconds")
	flagAuthUSER       = flag.String("user", "user", "user for smtp client")
	flagAuthPASS       = flag.String("pass", "pass", "pass for smtp client")
	flagDomain         = flag.String("domain", "localhost.com", "domain for recieving mails")
)

func init() {
	flag.Parse()
}
