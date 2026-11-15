package main

import "flag"

var (
	flagServerName     = flag.String("name", "smtp2http", "the server name")
	flagListenAddr     = flag.String("listen", ":smtp", "the smtp address to listen on")
	flagWebhook        = flag.String("webhook", "http://localhost:8080/my/webhook", "the webhook to send the data to")
	flagMaxMessageSize = flag.Int64("msglimit", 1024*1024*2, "maximum incoming message size")
	flagReadTimeout    = flag.Int("timeout.read", 5, "the read timeout in seconds")
	flagWriteTimeout   = flag.Int("timeout.write", 5, "the write timeout in seconds")
	flagWebhookTimeout = flag.Int("timeout.webhook", 30, "the webhook request timeout in seconds")
	flagDomain         = flag.String("domain", "", "only accept mail addressed to this domain (default: any)")

	flagLogLevel  = flag.String("log.level", "info", "log level: debug, info, warn or error")
	flagLogFormat = flag.String("log.format", "text", "log format: text or json")

	// Accepted but ignored. This server has never performed SMTP AUTH; the
	// flags exist so that pre-1.0 deployments passing them keep starting.
	// main() warns when either is set.
	flagAuthUSER = flag.String("user", "", "deprecated: ignored, this server does not authenticate senders")
	flagAuthPASS = flag.String("pass", "", "deprecated: ignored, this server does not authenticate senders")
)
