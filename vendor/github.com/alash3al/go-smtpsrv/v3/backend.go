package smtpsrv

import (
	"errors"

	"github.com/emersion/go-smtp"
)

// The Backend implements SMTP server methods.
type Backend struct {
	handler HandlerFunc
	auther  AuthFunc
}

func NewBackend(auther AuthFunc, handler HandlerFunc) *Backend {
	return &Backend{
		handler: handler,
		auther:  auther,
	}
}

// Login handles a login command with username and password.
func (bkd *Backend) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	if nil == bkd.auther {
		return nil, errors.New("invalid command specified")
	}

	return NewSession(state, bkd.handler, &username, &password), nil
}

// AnonymousLogin requires clients to authenticate using SMTP AUTH before sending emails
func (bkd *Backend) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	return NewSession(state, bkd.handler, nil, nil), nil
}
