package spf

import (
	"net"
	"sync"
)

// DNSResolver implements Resolver using local DNS
type DNSResolver struct{}

func errDNS(e error) error {
	if e == nil {
		return nil
	}
	if dnsErr, ok := e.(*net.DNSError); ok {
		// That is the most reliable way I found to detect Permerror
		// https://github.com/golang/go/blob/master/src/net/dnsclient.go#L43
		// Upon RCODE 3 return code we should return None result and pretend no
		//  From RFC7208:
		//  Several mechanisms rely on information fetched from the DNS.  For
		//  these DNS queries, except where noted, if the DNS server returns an
		//  error (RCODE other than 0 or 3) or the query times out, the
		//  mechanism stops and the topmost check_host() returns "temperror".
		//  If the server returns "Name Error" (RCODE 3), then evaluation of
		//  the mechanism continues as if the server returned no error (RCODE
		//  0) and zero answer records.
		if dnsErr.Err == "no such host" {
			return nil
		}
	}
	return ErrDNSTemperror
}

// LookupTXTStrict returns DNS TXT records for the given name, however it
// will return ErrDNSPermerror upon NXDOMAIN (RCODE 3)
func (r *DNSResolver) LookupTXTStrict(name string) ([]string, error) {
	txts, err := net.LookupTXT(name)

	if dnsErr, ok := err.(*net.DNSError); ok {
		// That is the most reliable way I found to detect Permerror
		// https://github.com/golang/go/blob/master/src/net/dnsclient.go#L43
		// Upon RCODE 3 return code we should return None result and pretend no
		//  From RFC7208:
		//  Several mechanisms rely on information fetched from the DNS.  For
		//  these DNS queries, except where noted, if the DNS server returns an
		//  error (RCODE other than 0 or 3) or the query times out, the
		//  mechanism stops and the topmost check_host() returns "temperror".
		//  If the server returns "Name Error" (RCODE 3), then evaluation of
		//  the mechanism continues as if the server returned no error (RCODE
		//  0) and zero answer records.
		if dnsErr.Err == "no such host" {
			return nil, ErrDNSPermerror
		}
	}

	err = errDNS(err)
	if err != nil {
		return nil, err
	}
	return txts, nil
}

// LookupTXT returns the DNS TXT records for the given domain name.
func (r *DNSResolver) LookupTXT(name string) ([]string, error) {
	txts, err := net.LookupTXT(name)
	err = errDNS(err)
	if err != nil {
		return nil, err
	}
	return txts, nil
}

// Exists is used for a DNS A RR lookup (even when the
// connection type is IPv6).  If any A record is returned, this
// mechanism matches.
func (r *DNSResolver) Exists(name string) (bool, error) {
	ips, err := net.LookupIP(name)
	err = errDNS(err)
	if err != nil {
		return false, err
	}
	return len(ips) > 0, nil
}

type hit struct {
	found bool
	err   error
}

// MatchIP provides an address lookup, which should be done on the name
// using the type of lookup (A or AAAA).
// Then IPMatcherFunc used to compare checked IP to the returned address(es).
// If any address matches, the mechanism matches
func (r *DNSResolver) MatchIP(name string, matcher IPMatcherFunc) (bool, error) {
	ips, err := net.LookupIP(name)
	err = errDNS(err)
	if err != nil {
		return false, err
	}
	for _, ip := range ips {
		if m, e := matcher(ip); m || e != nil {
			return m, e
		}
	}
	return false, nil
}

// MatchMX is similar to MatchIP but first performs an MX lookup on the
// name.  Then it performs an address lookup on each MX name returned.
// Then IPMatcherFunc used to compare checked IP to the returned address(es).
// If any address matches, the mechanism matches
func (r *DNSResolver) MatchMX(name string, matcher IPMatcherFunc) (bool, error) {
	mxs, err := net.LookupMX(name)
	err = errDNS(err)
	if err != nil {
		return false, err
	}

	var wg sync.WaitGroup
	hits := make(chan hit, len(mxs))

	for _, mx := range mxs {
		wg.Add(1)
		go func(name string) {
			found, err := r.MatchIP(name, matcher)
			hits <- hit{found, err}
			wg.Done()
		}(mx.Host)
	}

	go func() {
		wg.Wait()
		close(hits)
	}()

	for h := range hits {
		if h.found || h.err != nil {
			return h.found, h.err
		}
	}

	return false, nil
}
