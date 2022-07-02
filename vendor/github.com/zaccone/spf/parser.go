package spf

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

func matchingResult(qualifier tokenType) (Result, error) {
	switch qualifier {
	case qPlus:
		return Pass, nil
	case qMinus:
		return Fail, nil
	case qQuestionMark:
		return Neutral, nil
	case qTilde:
		return Softfail, nil
	default:
		return internalError, fmt.Errorf("invalid qualifier (%d)", qualifier) // TODO it's fishy; lexer must reject it before
	}
}

// SyntaxError represents parsing error, it holds reference to faulty token
// as well as error describing fault
type SyntaxError struct {
	token *token
	err   error
}

func (e SyntaxError) Error() string {
	return fmt.Sprintf("parse error for token %v: %v", e.token, e.err.Error())
}

// parser represents parsing structure. It keeps all arguments provided by top
// level CheckHost method as well as tokenized terms from TXT RR. One should
// call parser.Parse() for a proper SPF evaluation.
type parser struct {
	Sender      string
	Domain      string
	IP          net.IP
	Query       string
	Mechanisms  []*token
	Explanation *token
	Redirect    *token
	resolver    Resolver
}

// newParser creates new Parser objects and returns its reference.
// It accepts CheckHost() parameters as well as SPF query (fetched from TXT RR
// during initial DNS lookup.
func newParser(sender, domain string, ip net.IP, query string, resolver Resolver) *parser {
	return &parser{sender, domain, ip, query, make([]*token, 0, 10), nil, nil, resolver}
}

// parse aggregates all steps required for SPF evaluation.
// After lexing and tokenizing step it sorts tokens (and returns Permerror if
// there is any syntax error) and starts evaluating
// each token (from left to right). Once a token matches parse stops and
// returns matched result.
func (p *parser) parse() (Result, string, error) {
	tokens := lex(p.Query)

	if err := p.sortTokens(tokens); err != nil {
		return Permerror, "", err
	}

	var result = Neutral
	var matches bool
	var err error

	for _, token := range p.Mechanisms {
		switch token.mechanism {
		case tVersion:
			matches, result, err = p.parseVersion(token)
		case tAll:
			matches, result, err = p.parseAll(token)
		case tA:
			matches, result, err = p.parseA(token)
		case tIP4:
			matches, result, err = p.parseIP4(token)
		case tIP6:
			matches, result, err = p.parseIP6(token)
		case tMX:
			matches, result, err = p.parseMX(token)
		case tInclude:
			matches, result, err = p.parseInclude(token)
		case tExists:
			matches, result, err = p.parseExists(token)
		}

		if matches {
			if result == Fail && p.Explanation != nil {
				explanation, expError := p.handleExplanation()
				return result, explanation, expError
			}
			return result, "", err
		}

	}

	result, err = p.handleRedirect(Neutral)

	return result, "", err
}

func (p *parser) sortTokens(tokens []*token) error {
	all := false
	for _, token := range tokens {
		if token.mechanism.isErr() {
			return fmt.Errorf("syntax error for token: %v", token.value)
		} else if token.mechanism.isMechanism() && !all {
			p.Mechanisms = append(p.Mechanisms, token)

			if token.mechanism == tAll {
				all = true
			}
		} else {

			if token.mechanism == tRedirect {
				if p.Redirect == nil {
					p.Redirect = token
				} else {
					return errors.New(`too many "redirect"`)
				}
			} else if token.mechanism == tExp {
				if p.Explanation == nil {
					p.Explanation = token
				} else {
					return errors.New(`too many "exp"`)
				}
			}
		}
	}

	if all {
		p.Redirect = nil
	}

	return nil
}

func nonemptyString(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func (p *parser) parseVersion(t *token) (bool, Result, error) {
	if t.value == "spf1" {
		return false, None, nil
	}
	return true, Permerror, SyntaxError{t,
		fmt.Errorf("invalid spf qualifier: %v", t.value)}
}

func (p *parser) parseAll(t *token) (bool, Result, error) {
	result, err := matchingResult(t.qualifier)
	if err != nil {
		return true, Permerror, SyntaxError{t, err}
	}
	return true, result, nil

}

func (p *parser) parseIP4(t *token) (bool, Result, error) {
	result, _ := matchingResult(t.qualifier)

	if ip, ipnet, err := net.ParseCIDR(t.value); err == nil {
		if ip.To4() == nil {
			return true, Permerror, SyntaxError{t, errors.New("address isn't ipv4")}
		}
		return ipnet.Contains(p.IP), result, nil
	}

	ip := net.ParseIP(t.value).To4()
	if ip == nil {
		return true, Permerror, SyntaxError{t, errors.New("address isn't ipv4")}
	}
	return ip.Equal(p.IP), result, nil
}

func (p *parser) parseIP6(t *token) (bool, Result, error) {
	result, _ := matchingResult(t.qualifier)

	if ip, ipnet, err := net.ParseCIDR(t.value); err == nil {
		if ip.To16() == nil {
			return true, Permerror, SyntaxError{t, errors.New("address isn't ipv6")}
		}
		return ipnet.Contains(p.IP), result, nil

	}

	ip := net.ParseIP(t.value)
	if ip.To4() != nil || ip.To16() == nil {
		return true, Permerror, SyntaxError{t, errors.New("address isn't ipv6")}
	}
	return ip.Equal(p.IP), result, nil

}

func (p *parser) parseA(t *token) (bool, Result, error) {
	host, ip4Mask, ip6Mask, err := splitDomainDualCIDR(nonemptyString(t.value, p.Domain))
	if err != nil {
		return true, Permerror, SyntaxError{t, err}
	}

	result, _ := matchingResult(t.qualifier)

	found, err := p.resolver.MatchIP(NormalizeFQDN(host), func(ip net.IP) (bool, error) {
		n := net.IPNet{
			IP: ip,
		}
		switch len(ip) {
		case net.IPv4len:
			n.Mask = ip4Mask
		case net.IPv6len:
			n.Mask = ip6Mask
		}
		return n.Contains(p.IP), nil
	})
	return found, result, err
}

func (p *parser) parseMX(t *token) (bool, Result, error) {
	host, ip4Mask, ip6Mask, err := splitDomainDualCIDR(nonemptyString(t.value, p.Domain))
	if err != nil {
		return true, Permerror, SyntaxError{t, err}
	}

	result, _ := matchingResult(t.qualifier)
	found, err := p.resolver.MatchMX(NormalizeFQDN(host), func(ip net.IP) (bool, error) {
		n := net.IPNet{
			IP: ip,
		}
		switch len(ip) {
		case net.IPv4len:
			n.Mask = ip4Mask
		case net.IPv6len:
			n.Mask = ip6Mask
		}
		return n.Contains(p.IP), nil
	})
	return found, result, err
}

func (p *parser) parseInclude(t *token) (bool, Result, error) {
	domain := t.value
	if domain == "" {
		return true, Permerror, SyntaxError{t, errors.New("empty domain")}
	}
	theirResult, _, err := CheckHostWithResolver(p.IP, domain, p.Sender, p.resolver)

	/* Adhere to following result table:
	* +---------------------------------+---------------------------------+
	  | A recursive check_host() result | Causes the "include" mechanism  |
	  | of:                             | to:                             |
	  +---------------------------------+---------------------------------+
	  | pass                            | match                           |
	  |                                 |                                 |
	  | fail                            | not match                       |
	  |                                 |                                 |
	  | softfail                        | not match                       |
	  |                                 |                                 |
	  | neutral                         | not match                       |
	  |                                 |                                 |
	  | temperror                       | return temperror                |
	  |                                 |                                 |
	  | permerror                       | return permerror                |
	  |                                 |                                 |
	  | none                            | return permerror                |
	  +---------------------------------+---------------------------------+
	*/

	if err != nil {
		err = SyntaxError{t, err}
	}

	switch theirResult {
	case Pass:
		ourResult, _ := matchingResult(t.qualifier)
		return true, ourResult, err
	case Fail, Softfail, Neutral:
		return false, None, err
	case Temperror:
		return true, Temperror, err
	case None, Permerror:
		return true, Permerror, err
	default: // this should actually never happen
		return true, Permerror, SyntaxError{t, errors.New("unknown result")}
	}

}

func (p *parser) parseExists(t *token) (bool, Result, error) {
	resolvedDomain, err := parseMacroToken(p, t)
	if err != nil {
		return true, Permerror, SyntaxError{t, err}
	}
	if resolvedDomain == "" {
		return true, Permerror, SyntaxError{t, errors.New("empty domain")}
	}

	result, _ := matchingResult(t.qualifier)

	found, err := p.resolver.Exists(NormalizeFQDN(resolvedDomain))
	switch err {
	case nil:
		return found, result, nil
	case ErrDNSPermerror:
		return false, result, nil
	default:
		return false, Temperror, err // was true 8-|
	}
}

func (p *parser) handleRedirect(oldResult Result) (Result, error) {
	if p.Redirect == nil {
		return oldResult, nil
	}

	var (
		err    error
		result Result
	)

	redirectDomain := p.Redirect.value

	if result, _, err = CheckHostWithResolver(p.IP, redirectDomain, p.Sender, p.resolver); err != nil {
		//TODO(zaccone): confirm result value
		result = Permerror
	} else if result == None || result == Permerror {
		// See RFC7208, section 6.1
		//
		// if no SPF record is found, or if the <target-name> is malformed, the
		// result is a "permerror" rather than "none".
		result = Permerror
	}

	return result, err
}

func (p *parser) handleExplanation() (string, error) {
	domain, err := parseMacroToken(p, p.Explanation)
	if err != nil {
		return "", SyntaxError{p.Explanation, err}
	}
	if domain == "" {
		return "", SyntaxError{p.Explanation, errors.New("empty domain")}
	}

	txts, err := p.resolver.LookupTXT(NormalizeFQDN(domain))
	if err != nil {
		return "", err
	}

	// RFC 7208, section 6.2 specifies that result strings should be
	// concatenated with no spaces.
	exp, err := parseMacro(p, strings.Join(txts, ""))
	if err != nil {
		return "", SyntaxError{p.Explanation, err}
	}
	return exp, nil
}

func parseCIDRMask(s string, bits int) (net.IPMask, error) {
	if s == "" {
		return net.CIDRMask(bits, bits), nil
	}
	var (
		l   int
		err error
	)
	if l, err = strconv.Atoi(s); err != nil {
		return nil, errInvalidCIDRLength
	}
	mask := net.CIDRMask(l, bits)
	if mask == nil {
		return nil, errInvalidCIDRLength
	}
	return mask, nil
}

func splitDomainDualCIDR(domain string) (string, net.IPMask, net.IPMask, error) {
	var (
		ip4Mask net.IPMask
		ip6Mask net.IPMask
		ip4Len  string
		ip6Len  string
		err     error
	)

	parts := strings.SplitN(domain, "/", 3)
	domain = parts[0]
	if len(parts) > 1 {
		ip4Len = parts[1]
	}
	if len(parts) > 2 {
		ip6Len = parts[2]
	}

	if !isDomainName(domain) {
		return "", nil, nil, ErrInvalidDomain
	}
	ip4Mask, err = parseCIDRMask(ip4Len, 8*net.IPv4len)
	if err != nil {
		return "", nil, nil, err
	}
	ip6Mask, err = parseCIDRMask(ip6Len, 8*net.IPv6len)
	if err != nil {
		return "", nil, nil, err
	}

	return domain, ip4Mask, ip6Mask, nil
}
