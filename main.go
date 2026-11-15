package main

import (
	"encoding/base64"
	"flag"
	"io"
	"log/slog"
	"net/mail"
	"os"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/go-resty/resty/v2"
)

// One client for the process, not one per message: resty.New() per delivery
// threw away connection reuse to the webhook.
var webhookClient = resty.New()

func main() {
	// not init(): flag.Parse() there also eats `go test` flags and breaks the tests
	flag.Parse()

	slog.SetDefault(newLogger(*flagLogLevel, *flagLogFormat))

	if *flagAuthUSER != "" || *flagAuthPASS != "" {
		slog.Warn("-user/-pass are accepted for compatibility but ignored; " +
			"this server does not authenticate senders. Restrict access with -domain and your firewall")
	}

	// Without a timeout a webhook that accepts the connection and then stalls
	// holds the SMTP session, and the message it buffered, open forever.
	webhookClient.SetTimeout(time.Duration(*flagWebhookTimeout) * time.Second)

	srv := smtp.NewServer(&backend{handle: deliver})
	srv.Addr = *flagListenAddr
	srv.Domain = *flagServerName
	srv.ReadTimeout = time.Duration(*flagReadTimeout) * time.Second
	srv.WriteTimeout = time.Duration(*flagWriteTimeout) * time.Second
	srv.MaxMessageBytes = *flagMaxMessageSize
	// One webhook POST carries one recipient. Further RCPT TOs get a 452, which
	// makes the sender deliver them in transactions of their own; accepting them
	// here would answer 250 and then deliver only the last one.
	srv.MaxRecipients = 1
	srv.AllowInsecureAuth = true
	srv.EnableSMTPUTF8 = false

	slog.Info("starting",
		"webhook", redactURL(*flagWebhook),
		"domain_filter", domainFilter(),
		"msglimit", *flagMaxMessageSize)

	if err := listenAndServe(srv); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

// domainFilter renders -domain for the startup line, so an operator can see at
// a glance that the default accepts mail for every domain.
func domainFilter() string {
	if *flagDomain == "" {
		return "(any)"
	}

	return *flagDomain
}

// SMTP replies. A handler error that is not an *smtp.SMTPError is answered
// with a permanent 554 by go-smtp, which makes the sending server bounce the
// mail at once. A webhook that is down or erroring is our problem, not the
// sender's, so those are 451: the sender keeps the message queued and retries.
var (
	errDomainNotAllowed = &smtp.SMTPError{
		Code:         550,
		EnhancedCode: smtp.EnhancedCode{5, 7, 1},
		Message:      "Unauthorized TO domain",
	}
	errWebhookUnreachable = &smtp.SMTPError{
		Code:         451,
		EnhancedCode: smtp.EnhancedCode{4, 3, 0},
		Message:      "E1: Cannot accept your message due to internal error, please report that to our engineers",
	}
	errWebhookStatus = &smtp.SMTPError{
		Code:         451,
		EnhancedCode: smtp.EnhancedCode{4, 3, 0},
		Message:      "E2: Cannot accept your message due to internal error, please report that to our engineers",
	}
)

// deliver POSTs one message to the webhook. A webhook failure is a temporary
// SMTP error, so the sender retries instead of the mail being lost.
func deliver(env *envelope) error {
	started := time.Now()

	log := slog.With(
		"from", env.from.Address,
		"to", env.to.Address,
		"message_id", env.message.MessageID,
		"spf", string(env.spf),
	)

	if !recipientAllowed(env.to.Address, *flagDomain) {
		log.Warn("rejected", "reason", "to_domain_not_allowed", "domain_filter", domainFilter())
		return errDomainNotAllowed
	}

	payload := buildPayload(env)

	resp, err := webhookClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(payload).
		Post(*flagWebhook)
	if err != nil {
		// Transport-level failure: the webhook was unreachable, refused the
		// connection or timed out.
		log.Error("rejected", "reason", "webhook_unreachable", "err", redactErr(err),
			"duration_ms", time.Since(started).Milliseconds())

		return errWebhookUnreachable
	}

	if resp.StatusCode() != 200 {
		log.Error("rejected", "reason", "webhook_status", "status", resp.StatusCode(),
			"duration_ms", time.Since(started).Milliseconds())

		return errWebhookStatus
	}

	log.Info("accepted",
		"subject_len", len(env.message.Subject),
		"attachments", len(payload.Attachments),
		"embedded_files", len(payload.EmbeddedFiles),
		"duration_ms", time.Since(started).Milliseconds())

	return nil
}

// buildPayload maps a parsed message onto the JSON sent to the webhook. The
// shape is a public contract -- testdata/golden holds the payloads 1.0.0
// produced, and parser_test.go checks this still matches them.
func buildPayload(env *envelope) *EmailMessage {
	msg := env.message

	out := EmailMessage{
		ID:            msg.MessageID,
		Date:          msg.Date.String(),
		References:    msg.References,
		SPFResult:     string(env.spf),
		ResentDate:    msg.ResentDate.String(),
		ResentID:      msg.ResentMessageID,
		Subject:       msg.Subject,
		Attachments:   []*EmailAttachment{},
		EmbeddedFiles: []*EmailEmbeddedFile{},
	}

	out.Body.HTML = string(msg.HTMLBody)
	out.Body.Text = string(msg.TextBody)

	out.Addresses.From = transformStdAddressToEmailAddress([]*mail.Address{env.from})[0]
	out.Addresses.To = transformStdAddressToEmailAddress([]*mail.Address{env.to})[0]
	out.Addresses.Cc = transformStdAddressToEmailAddress(msg.Cc)
	out.Addresses.Bcc = transformStdAddressToEmailAddress(msg.Bcc)
	out.Addresses.ReplyTo = transformStdAddressToEmailAddress(msg.ReplyTo)
	out.Addresses.InReplyTo = msg.InReplyTo

	if resentFrom := transformStdAddressToEmailAddress(msg.ResentFrom); len(resentFrom) > 0 {
		out.Addresses.ResentFrom = resentFrom[0]
	}

	out.Addresses.ResentTo = transformStdAddressToEmailAddress(msg.ResentTo)
	out.Addresses.ResentCc = transformStdAddressToEmailAddress(msg.ResentCc)
	out.Addresses.ResentBcc = transformStdAddressToEmailAddress(msg.ResentBcc)

	for _, a := range msg.Attachments {
		data, _ := io.ReadAll(a.Data)
		out.Attachments = append(out.Attachments, &EmailAttachment{
			Filename:    a.Filename,
			ContentType: a.ContentType,
			Data:        base64.StdEncoding.EncodeToString(data),
		})
	}

	for _, a := range msg.EmbeddedFiles {
		data, _ := io.ReadAll(a.Data)
		out.EmbeddedFiles = append(out.EmbeddedFiles, &EmailEmbeddedFile{
			CID:         a.CID,
			ContentType: a.ContentType,
			Data:        base64.StdEncoding.EncodeToString(data),
		})
	}

	return &out
}
