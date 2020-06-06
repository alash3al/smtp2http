package main

import (
	"fmt"

	"github.com/alash3al/go-smtpsrv"
)

func main() {
	srv := &smtpsrv.Server{
		Addr:        *flagListenAddr,
		MaxBodySize: *flagMaxMessageSize,
		Handler:     handler,
		Name:        *flagServerName,
	}

	fmt.Println("start the smtp server on address: ", *flagListenAddr)
	fmt.Println("specified maximum body size: ", *flagMaxMessageSize, " bytes")
	fmt.Println("specified server name: ", *flagServerName)
	fmt.Println("specified webhook: ", *flagWebhook)
	fmt.Println("validating the incoming FROM header: ", *flagStrictValidation)

	fmt.Println(srv.ListenAndServe())
}
