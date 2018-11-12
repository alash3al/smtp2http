package main

import "flag"

var (
	flagServerName       = flag.String("name", "smtp2http", "the server name")
	flagListenAddr       = flag.String("listen", ":smtp", "the smtp address to listen on")
	flagWebhook          = flag.String("webhook", "http://localhost:8080/my/webhook", "the webhook to send the data to")
	flagMaxMessageSize   = flag.Int64("msglimit", 1024*1024*2, "maximum incoming message size")
	flagStrictValidation = flag.Bool("strict", true, "strict validation including spf, host, format, user and messageID")
)

func init() {
	flag.Parse()
}
