package spf

import "strings"

// addrSpec abstracts Addr-Spec as it defined by RFC5322
// https://tools.ietf.org/html/rfc5322#section-3.4.1
type addrSpec struct {
	local  string
	domain string
}

// parseAddrSpec parses e-mail string address and returns *addrSpec structure.
// The "postmaster" will be used if no local part specified in addr.
// The domain will be used if no domain specified in addr.
func parseAddrSpec(addr, domain string) *addrSpec {
	const postmaster string = "postmaster"

	if addr == "" || addr == "@" {
		return &addrSpec{postmaster, domain}
	}

	var l, d string
	i := strings.LastIndexByte(addr, '@')
	if i < 0 || i == len(addr)-1 { // local[@]
		d = domain
	} else {
		d = addr[i+1:]
	}
	if i == 0 { // @domain
		l = postmaster
	} else {
		l = addr[:i]
	}

	return &addrSpec{l, d}
}
