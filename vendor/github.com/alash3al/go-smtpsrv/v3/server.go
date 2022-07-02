package smtpsrv

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/emersion/go-smtp"
)

type ServerConfig struct {
	ListenAddr      string
	BannerDomain    string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	Handler         HandlerFunc
	Auther          AuthFunc
	MaxMessageBytes int
	TLSConfig       *tls.Config
}

func ListenAndServe(cfg *ServerConfig) error {
	s := smtp.NewServer(NewBackend(cfg.Auther, cfg.Handler))

	SetDefaultServerConfig(cfg)

	s.Addr = cfg.ListenAddr
	s.Domain = cfg.BannerDomain
	s.ReadTimeout = cfg.ReadTimeout
	s.WriteTimeout = cfg.WriteTimeout
	s.MaxMessageBytes = cfg.MaxMessageBytes
	s.AllowInsecureAuth = true
	s.AuthDisabled = true
	s.EnableSMTPUTF8 = false

	fmt.Println("⇨ smtp server started on", s.Addr)

	return s.ListenAndServe()
}

func ListenAndServeTLS(cfg *ServerConfig) error {
	s := smtp.NewServer(NewBackend(cfg.Auther, cfg.Handler))

	SetDefaultServerConfig(cfg)

	s.Addr = cfg.ListenAddr
	s.Domain = cfg.BannerDomain
	s.ReadTimeout = cfg.ReadTimeout
	s.WriteTimeout = cfg.WriteTimeout
	s.MaxMessageBytes = cfg.MaxMessageBytes
	s.AllowInsecureAuth = true
	s.AuthDisabled = true
	s.EnableSMTPUTF8 = false
	s.EnableREQUIRETLS = true
	s.TLSConfig = cfg.TLSConfig

	fmt.Println("⇨ smtp server started on", s.Addr)

	return s.ListenAndServeTLS()
}
