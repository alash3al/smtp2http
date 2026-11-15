package main

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/smtp"
	"net/textproto"
	"strings"
	"testing"

	gosmtp "github.com/emersion/go-smtp"
)

// startServer runs the real SMTP backend on a loopback port and points the
// webhook flag at hook for the duration of the test.
func startServer(t *testing.T, hook http.HandlerFunc, domain string) string {
	t.Helper()

	webhook := httptest.NewServer(hook)
	t.Cleanup(webhook.Close)

	oldWebhook, oldDomain := *flagWebhook, *flagDomain
	*flagWebhook, *flagDomain = webhook.URL, domain
	t.Cleanup(func() { *flagWebhook, *flagDomain = oldWebhook, oldDomain })

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	srv := gosmtp.NewServer(&backend{handle: deliver})
	srv.Domain = "test"
	srv.MaxRecipients = 1
	srv.AllowInsecureAuth = true

	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	return ln.Addr().String()
}

// send delivers one message and returns the error from the end of DATA, which
// is where the server reports what became of it.
func send(t *testing.T, addr, to, raw string) error {
	t.Helper()

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	if err := c.Mail("alice@sender.invalid"); err != nil {
		t.Fatal(err)
	}

	if err := c.Rcpt(to); err != nil {
		t.Fatal(err)
	}

	w, err := c.Data()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := io.WriteString(w, raw); err != nil {
		t.Fatal(err)
	}

	return w.Close()
}

const plainMessage = "From: Alice <alice@sender.invalid>\r\n" +
	"To: Bob <bob@example.com>\r\n" +
	"Subject: hello\r\n" +
	"Content-Type: text/plain; charset=utf-8\r\n" +
	"Content-Transfer-Encoding: 8bit\r\n" +
	"\r\n" +
	"Dobrý deň\r\n"

func TestSMTPDeliversToWebhook(t *testing.T) {
	got := make(chan EmailMessage, 1)

	addr := startServer(t, func(w http.ResponseWriter, r *http.Request) {
		var msg EmailMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			t.Errorf("webhook body: %v", err)
		}

		got <- msg
	}, "")

	if err := send(t, addr, "bob@example.com", plainMessage); err != nil {
		t.Fatalf("message was not accepted: %v", err)
	}

	msg := <-got
	if msg.Subject != "hello" || !strings.HasPrefix(msg.Body.Text, "Dobrý deň") {
		t.Errorf("payload = subject %q, text %q", msg.Subject, msg.Body.Text)
	}

	if msg.Addresses.From.Address != "alice@sender.invalid" || msg.Addresses.To.Address != "bob@example.com" {
		t.Errorf("envelope = %+v -> %+v", msg.Addresses.From, msg.Addresses.To)
	}
}

// A failing webhook must be a temporary error. A permanent one makes the
// sending server bounce the mail instead of retrying it.
func TestSMTPReplyCodes(t *testing.T) {
	cases := []struct {
		name     string
		status   int
		domain   string
		raw      string
		wantCode int
	}{
		{"webhook 500 is temporary", 500, "", plainMessage, 451},
		{"webhook 401 is temporary", 401, "", plainMessage, 451},
		{"wrong domain is permanent", 200, "other.tld", plainMessage, 550},
		{"unparseable message is permanent", 200, "", "no headers at all\r\n", 554},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			addr := startServer(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(c.status)
			}, c.domain)

			err := send(t, addr, "bob@example.com", c.raw)

			tperr, ok := err.(*textproto.Error)
			if !ok {
				t.Fatalf("want an SMTP error, got %v", err)
			}

			if tperr.Code != c.wantCode {
				t.Errorf("reply = %d %s, want %d", tperr.Code, tperr.Msg, c.wantCode)
			}
		})
	}
}

func TestSMTPWebhookDownIsTemporary(t *testing.T) {
	addr := startServer(t, func(http.ResponseWriter, *http.Request) {}, "")

	// Nothing listens on port 1.
	*flagWebhook = "http://127.0.0.1:1/hook"

	err := send(t, addr, "bob@example.com", plainMessage)

	if tperr, ok := err.(*textproto.Error); !ok || tperr.Code != 451 {
		t.Errorf("reply = %v, want 451", err)
	}
}

// A second RCPT TO must be refused with a temporary error so the sender
// delivers it separately. Accepting it would lose every recipient but the last.
func TestSMTPSecondRecipientIsDeferred(t *testing.T) {
	addr := startServer(t, func(http.ResponseWriter, *http.Request) {}, "")

	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	if err := c.Mail("alice@sender.invalid"); err != nil {
		t.Fatal(err)
	}

	if err := c.Rcpt("bob@example.com"); err != nil {
		t.Fatal(err)
	}

	err = c.Rcpt("carol@example.com")
	if tperr, ok := err.(*textproto.Error); !ok || tperr.Code != 452 {
		t.Errorf("second RCPT = %v, want 452", err)
	}
}
