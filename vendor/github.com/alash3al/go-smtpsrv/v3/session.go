package smtpsrv

import (
	"errors"
	"io"
	"net/mail"

	"github.com/emersion/go-smtp"
)

// A Session is returned after successful login.
type Session struct {
	connState *smtp.ConnectionState
	From      *mail.Address
	To        *mail.Address
	handler   HandlerFunc
	body      io.Reader
	username  *string
	password  *string
}

// NewSession initialize a new session
func NewSession(state *smtp.ConnectionState, handler HandlerFunc, username, password *string) *Session {
	return &Session{
		connState: state,
		handler:   handler,
	}
}

func (s *Session) Mail(from string, opts smtp.MailOptions) (err error) {
	s.From, err = mail.ParseAddress(from)
	return
}

func (s *Session) Rcpt(to string) (err error) {
	s.To, err = mail.ParseAddress(to)
	return
}

func (s *Session) Data(r io.Reader) error {
	if s.handler == nil {
		return errors.New("internal error: no handler")
	}

	s.body = r

	c := Context{
		session: s,
	}

	return s.handler(&c)
}

func (s *Session) Reset() {
}

func (s *Session) Logout() error {
	return nil
}
