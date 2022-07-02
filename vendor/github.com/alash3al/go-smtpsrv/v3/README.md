A SMTP Server Package [![](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square)](https://godoc.org/github.com/alash3al/go-smtpsrv)
=============================
a simple smtp server library, a simple wrapper for a well-maintained package called [go-smtp](github.com/emersion/go-smtp).

Quick Start
===========
> `go get github.com/alash3al/go-smtpsrv`

```go
package main

import (
	"fmt"

	"github.com/alash3al/go-smtpsrv"
)

func main() {
	handler := func(c smtpsrv.Context) error {
		// ...
		return nil
	}

	cfg := smtpsrv.ServerConfig{
		BannerDomain:  "mail.my.server",
		ListenAddress: ":25025",
		MaxMessageBytes: 5 * 1024,
		Handler:     handler,
	}

	fmt.Println(smtpsrv.ListenAndServe(cfg))
}

```
