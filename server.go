package main

import (
	"errors"
	"io"
	"log/slog"
	"net"
	"net/mail"
	"strings"

	"blitiri.com.ar/go/spf"
	"github.com/emersion/go-smtp"
)

// handler processes one fully received message. Returning an error fails the
// SMTP transaction, which tells the sending server to retry rather than drop
// the mail on the floor.
type handler func(env *envelope) error

// envelope is what the handler needs: the SMTP-level addresses (which are not
// necessarily the From/To headers), the parsed message, and the SPF verdict.
type envelope struct {
	from    *mail.Address
	to      *mail.Address
	message *Email
	spf     spf.Result
}

type backend struct {
	handle handler
}

func (b *backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	var remote net.IP
	if addr, ok := c.Conn().RemoteAddr().(*net.TCPAddr); ok {
		remote = addr.IP
	}

	return &session{handle: b.handle, remote: remote}, nil
}

type session struct {
	handle handler
	remote net.IP
	from   string
	to     string
}

func (s *session) Mail(from string, _ *smtp.MailOptions) error {
	s.from = from
	return nil
}

func (s *session) Rcpt(to string, _ *smtp.RcptOptions) error {
	s.to = to
	return nil
}

func (s *session) Data(r io.Reader) error {
	msg, err := ParseEmail(r)
	if err != nil {
		// Permanent: the same bytes will not parse on a retry either.
		return &smtp.SMTPError{
			Code:         554,
			EnhancedCode: smtp.EnhancedCode{5, 6, 0},
			Message:      "Cannot read your message: " + err.Error(),
		}
	}

	from, err := parseSMTPAddress(s.from)
	if err != nil {
		return err
	}

	to, err := parseSMTPAddress(s.to)
	if err != nil {
		return err
	}

	return s.handle(&envelope{
		from:    from,
		to:      to,
		message: msg,
		spf:     s.checkSPF(),
	})
}

// checkSPF mirrors what go-smtpsrv did: evaluate the envelope sender against
// the connecting IP, and treat any failure to evaluate as "none" rather than
// rejecting the message.
func (s *session) checkSPF() spf.Result {
	if s.remote == nil || s.from == "" {
		return spf.None
	}

	at := strings.LastIndex(s.from, "@")
	if at < 0 {
		return spf.None
	}

	result, _ := spf.CheckHostWithSender(s.remote, s.from[at+1:], s.from)

	return result
}

func (s *session) Reset() { s.from, s.to = "", "" }

func (s *session) Logout() error { return nil }

// parseSMTPAddress accepts a bare envelope address. go-smtp hands over the
// address without angle brackets, but net/mail wants a full address, so an
// unparseable value falls back to using it verbatim.
func parseSMTPAddress(addr string) (*mail.Address, error) {
	if addr == "" {
		return nil, errors.New("missing address")
	}

	if parsed, err := mail.ParseAddress(addr); err == nil {
		return parsed, nil
	}

	return &mail.Address{Address: addr}, nil
}

// listenAndServe binds first so the log line can report the address the kernel
// actually gave us. go-smtpsrv printed the raw config string, so the default
// ":smtp" was reported verbatim and never as port 25.
func listenAndServe(cfg *smtp.Server) error {
	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return err
	}

	slog.Info("listening", "addr", ln.Addr().String())

	return cfg.Serve(ln)
}
