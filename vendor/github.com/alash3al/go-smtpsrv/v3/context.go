package smtpsrv

import (
	"crypto/tls"
	"net"
	"net/mail"

	"github.com/zaccone/spf"
)

type Context struct {
	session *Session
}

func (c Context) From() *mail.Address {
	return c.session.From
}

func (c Context) To() *mail.Address {
	return c.session.To
}

func (c Context) User() (string, string, error) {
	if c.session.username == nil || c.session.password == nil {
		return "", "", ErrAuthDisabled
	}

	return *c.session.username, *c.session.password, nil
}

func (c Context) RemoteAddr() net.Addr {
	return c.session.connState.RemoteAddr
}

func (c Context) TLS() *tls.ConnectionState {
	return &c.session.connState.TLS
}

func (c Context) Read(p []byte) (int, error) {
	return c.session.body.Read(p)
}

func (c Context) Parse() (*Email, error) {
	return ParseEmail(c.session.body)
}

func (c Context) Mailable() (bool, error) {
	_, host, err := SplitAddress(c.From().Address)
	if err != nil {
		return false, err
	}

	mxhosts, err := net.LookupMX(host)
	if err != nil {
		return false, err
	}

	return len(mxhosts) > 0, nil
}
func (c Context) SPF() (SPFResult, string, error) {
	_, host, err := SplitAddress(c.From().Address)
	if err != nil {
		return spf.None, "", err
	}

	return spf.CheckHost(net.ParseIP(c.RemoteAddr().String()), host, c.From().Address)
}
