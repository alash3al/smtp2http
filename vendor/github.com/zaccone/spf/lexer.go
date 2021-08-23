package spf

import (
	"strings"
	"unicode/utf8"
)

// lexer represents lexing structure
type lexer struct {
	start  int
	pos    int
	prev   int
	length int
	input  string
}

// lex reads SPF record and returns list of Tokens along with
// their modifiers and values. Parser should parse the Tokens and execute
// relevant actions
func lex(input string) []*token {
	var tokens []*token
	l := &lexer{0, 0, 0, len(input), input}
	for {
		token := l.scan()
		if token.mechanism == tEOF {
			break
		}
		tokens = append(tokens, token)
	}
	return tokens
}

// scan scans input and returns a Token structure
func (l *lexer) scan() *token {
	for {
		r, eof := l.next()
		if eof {
			return &token{tEOF, tEOF, ""}
		} else if isWhitespace(r) || l.eof() { // we just scanned some meaningful data
			token := l.scanIdent()
			l.scanWhitespaces()
			l.moveon()
			return token
		}
	}
}

// Lexer.eof() return true when scanned record has ended, false otherwise
func (l *lexer) eof() bool { return l.pos >= l.length }

// Lexer.next() returns next read rune and boolean indicator whether scanned
// record has ended. Method also moves `pos` value to size (length of read rune),
// and `prev` to previous `pos` location.
func (l *lexer) next() (rune, bool) {
	if l.eof() {
		return 0, true
	}
	r, size := utf8.DecodeRuneInString(l.input[l.pos:])
	// TODO(zaccone): check for operation success/failure
	l.prev = l.pos
	l.pos += size
	return r, false
}

// Lexer.moveon() sets Lexer.start to Lexer.pos. This is usually done once the
// ident has been scanned.
func (l *lexer) moveon() { l.start = l.pos }

// Lexer.back() moves back current Lexer.pos to a previous position.
func (l *lexer) back() { l.pos = l.prev }

// scanWhitespaces moves position to a first rune which is not a
// whitespace or tab
func (l *lexer) scanWhitespaces() {
	for {
		if ch, eof := l.next(); eof {
			return
		} else if !isWhitespace(ch) {
			l.back()
			return
		}
	}
}

// scanIdent is a Lexer method executed after an ident was found.
// It operates on a slice with constraints [l.start:l.pos).
// A cursor tries to find delimiters and set proper `mechanism`, `qualifier`
// and value itself.
// The default token has `mechanism` set to tErr, that is, error state.
func (l *lexer) scanIdent() *token {
	t := &token{tErr, qPlus, ""}
	cursor := l.start
	for cursor < l.pos {
		ch, size := utf8.DecodeRuneInString(l.input[cursor:])
		cursor += size

		if isQualifier(ch) {
			t.qualifier, _ = qualifiers[ch]
			l.start = cursor
			continue
		} else if isDelimiter(ch) { // add error handling
			t.mechanism = tokenTypeFromString(l.input[l.start : cursor-size])
			t.value = strings.TrimSpace(l.input[cursor:l.pos])

			if t.value == "" || !checkTokenSyntax(t, ch) {
				t.qualifier = qErr
				t.mechanism = tErr
			}

			break
		}
	}

	if t.mechanism.isErr() {
		t.mechanism = tokenTypeFromString(
			strings.TrimSpace(l.input[l.start:cursor]))
		if t.mechanism.isErr() {
			t.qualifier = qErr
			t.value = ""
		}
	}
	return t
}

// isWhitespace returns true if the rune is a space, tab, or newline.
func isWhitespace(ch rune) bool { return ch == ' ' || ch == '\t' || ch == '\n' }

// isDelimiter returns true if rune equals to ':' or '=', false otherwise
func isDelimiter(ch rune) bool { return ch == ':' || ch == '=' }

// isQualifier returns true if rune is a SPF delimiter (+,-,!,?)
func isQualifier(ch rune) bool { return ch == '+' || ch == '-' || ch == '~' || ch == '?' }

// isDigit returns true if rune is a numer (between '0' and '9'), false otherwise
func isDigit(ch rune) bool { return ch >= '0' && ch <= '9' }
