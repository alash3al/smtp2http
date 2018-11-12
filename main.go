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
	fmt.Println(srv.ListenAndServe())
}
