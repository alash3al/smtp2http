package spf

import "strconv"

type tokenType int

const (
	tEOF tokenType = iota
	tErr

	mechanismBeg

	tVersion // used only for v=spf1 starter
	tAll     // all
	tA       // a
	tIP4     // ip4
	tIP6     // ip6
	tMX      // mx
	tPTR     // ptr
	tInclude // include
	tExists  // exists

	mechanismEnd

	modifierBeg

	tRedirect // redirect
	tExp      // explanation

	modifierEnd

	_ // qEmpty - deadcode, not used
	qPlus
	qMinus
	qTilde
	qQuestionMark

	qErr
)

var qualifiers = map[rune]tokenType{
	'+': qPlus,
	'-': qMinus,
	'?': qQuestionMark,
	'~': qTilde,
}

func (tok tokenType) String() string {
	switch tok {
	case tVersion:
		return "v"
	case tAll:
		return "all"
	case tIP4:
		return "ip4"
	case tIP6:
		return "ip6"
	case tMX:
		return "mx"
	case tPTR:
		return "ptr"
	case tInclude:
		return "include"
	case tRedirect:
		return "redirect"
	case tExists:
		return "exists"
	case tExp:
		return "exp"
	default:
		return strconv.Itoa(int(tok))
	}
}

func tokenTypeFromString(s string) tokenType {
	switch s {
	case "v":
		return tVersion
	case "all":
		return tAll
	case "a":
		return tA
	case "ip4":
		return tIP4
	case "ip6":
		return tIP6
	case "mx":
		return tMX
	case "ptr":
		return tPTR
	case "include":
		return tInclude
	case "redirect":
		return tRedirect
	case "exists":
		return tExists
	case "explanation", "exp":
		return tExp
	default:
		return tErr
	}
}

func (tok tokenType) isErr() bool { return tok == tErr }

func (tok tokenType) isMechanism() bool {
	return tok > mechanismBeg && tok < mechanismEnd
}

func (tok tokenType) isModifier() bool {
	return tok > modifierBeg && tok < modifierEnd
}

func checkTokenSyntax(tkn *token, delimiter rune) bool {
	if tkn == nil {
		return false
	}

	if tkn.mechanism == tErr && tkn.qualifier == qErr {
		return true // syntax is ok
	}

	// special case for v=spf1 token

	if tkn.mechanism == tVersion {
		return true
	}

	//mechanism include must not have empty content
	if tkn.mechanism == tInclude && tkn.value == "" {
		return false
	}
	if tkn.mechanism.isModifier() && delimiter != '=' {
		return false
	}
	if tkn.mechanism.isMechanism() && delimiter != ':' {
		return false
	}

	return true
}

// token represents SPF term (modifier or mechanism) like all, include, a, mx,
// ptr, ip4, ip6, exists, redirect etc.
// It's a base structure later parsed by Parser.
type token struct {
	mechanism tokenType // all, include, a, mx, ptr, ip4, ip6, exists etc.
	qualifier tokenType // +, -, ~, ?, defaults to +
	value     string    // value for a mechanism
}
