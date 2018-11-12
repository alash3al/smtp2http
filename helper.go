package main

import (
	"net/mail"
)

func extractEmails(addr []*mail.Address, _ ...error) []string {
	ret := []string{}

	for _, e := range addr {
		ret = append(ret, e.Address)
	}

	return ret
}
