package smtpsrv

import (
	"errors"
	"strings"
	"time"
)

// SplitAddress split the email@addre.ss to <user>@<domain>
func SplitAddress(address string) (string, string, error) {
	sepInd := strings.LastIndex(address, "@")
	if sepInd == -1 {
		return "", "", errors.New("Invalid Address:" + address)
	}
	localPart := address[:sepInd]
	domainPart := address[sepInd+1:]
	return localPart, domainPart, nil
}

func SetDefaultServerConfig(cfg *ServerConfig) {
	if cfg == nil {
		*cfg = ServerConfig{}
	}

	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "[::]:25025"
	}

	if cfg.BannerDomain == "" {
		cfg.BannerDomain = "localhost"
	}

	if cfg.ReadTimeout < 1 {
		cfg.ReadTimeout = 2 * time.Second
	}

	if cfg.WriteTimeout < 1 {
		cfg.WriteTimeout = 2 * time.Second
	}

	if cfg.MaxMessageBytes < 1 {
		cfg.MaxMessageBytes = 1024 * 1024 * 2
	}
}
