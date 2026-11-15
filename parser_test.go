package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"blitiri.com.ar/go/spf"
)

// TestGoldenPayloads feeds every corpus message through the parser and payload
// builder and compares the result to the payload the published 1.0.0 image
// produced for it. The webhook JSON is a public contract; this is what makes
// swapping the SMTP server, parser and SPF library reviewable.
//
// Regenerate the goldens with testdata/capture.py -- see testdata/README.md.
func TestGoldenPayloads(t *testing.T) {
	files, err := filepath.Glob("testdata/corpus/*.eml")
	if err != nil || len(files) == 0 {
		t.Fatalf("no corpus files: %v", err)
	}

	for _, path := range files {
		name := strings.TrimSuffix(filepath.Base(path), ".eml")

		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			msg, err := ParseEmail(bytes.NewReader(raw))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			// The envelope addresses capture.py used, and spf.None because
			// sender.invalid publishes no SPF record. Keeping the verdict an
			// input rather than a lookup keeps this test off the network.
			got := buildPayload(&envelope{
				from:    &mail.Address{Address: "alice@sender.invalid"},
				to:      &mail.Address{Address: "bob@example.com"},
				message: msg,
				spf:     spf.None,
			})

			want, err := os.ReadFile(filepath.Join("testdata/golden", name+".json"))
			if err != nil {
				t.Fatalf("golden: %v", err)
			}

			if diff := jsonDiff(t, want, got); diff != "" {
				t.Errorf("payload changed vs 1.0.0:\n%s", diff)
			}
		})
	}
}

// jsonDiff compares two payloads structurally, so key order and formatting do
// not matter -- only the content does.
func jsonDiff(t *testing.T, wantRaw []byte, got any) string {
	t.Helper()

	gotRaw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}

	var wantAny, gotAny any
	if err := json.Unmarshal(wantRaw, &wantAny); err != nil {
		t.Fatal(err)
	}

	if err := json.Unmarshal(gotRaw, &gotAny); err != nil {
		t.Fatal(err)
	}

	wantNorm, _ := json.MarshalIndent(wantAny, "", "  ")
	gotNorm, _ := json.MarshalIndent(gotAny, "", "  ")

	if bytes.Equal(wantNorm, gotNorm) {
		return ""
	}

	return "--- want (1.0.0)\n" + string(wantNorm) + "\n+++ got\n" + string(gotNorm)
}

// FuzzParseEmail exercises the vendored parser against arbitrary input. It now
// lives in this repository, and it is the component that touches untrusted
// bytes first, so it should not panic on anything.
func FuzzParseEmail(f *testing.F) {
	files, _ := filepath.Glob("testdata/corpus/*.eml")
	for _, path := range files {
		if raw, err := os.ReadFile(path); err == nil {
			f.Add(raw)
		}
	}

	f.Fuzz(func(t *testing.T, raw []byte) {
		// Errors are expected and fine; panics are not.
		_, _ = ParseEmail(bytes.NewReader(raw))
	})
}

// Messages 1.0.0 rejected outright or passed through undecoded, so there is no
// golden payload to compare with: these assert the decoded bodies directly.
func TestParseTransferEncodings(t *testing.T) {
	cases := []struct {
		name, raw, wantText, wantHTML string
	}{
		{
			name: "8bit",
			raw: "From: a@b.c\r\nContent-Type: text/plain; charset=utf-8\r\n" +
				"Content-Transfer-Encoding: 8bit\r\n\r\nDobrý deň\r\n",
			wantText: "Dobrý deň\r",
		},
		{
			name: "binary",
			raw: "From: a@b.c\r\nContent-Type: text/plain\r\n" +
				"Content-Transfer-Encoding: binary\r\n\r\nhello\r\n",
			wantText: "hello\r",
		},
		{
			name: "encoding name in another case",
			raw: "From: a@b.c\r\nContent-Type: text/plain\r\n" +
				"Content-Transfer-Encoding: Base64\r\n\r\naGVsbG8=\r\n",
			wantText: "hello",
		},
		{
			name: "8bit part inside multipart/alternative",
			raw: "From: a@b.c\r\nContent-Type: multipart/alternative; boundary=X\r\n\r\n" +
				"--X\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\nDobrý deň\r\n" +
				"--X\r\nContent-Type: text/html; charset=utf-8\r\nContent-Transfer-Encoding: 8BIT\r\n\r\n<p>Dobrý deň</p>\r\n" +
				"--X--\r\n",
			wantText: "Dobrý deň",
			wantHTML: "<p>Dobrý deň</p>",
		},
		{
			name: "base64 html inside multipart/related",
			raw: "From: a@b.c\r\nContent-Type: multipart/related; boundary=X\r\n\r\n" +
				"--X\r\nContent-Type: text/html; charset=utf-8\r\nContent-Transfer-Encoding: base64\r\n\r\n" +
				"PHA+RG9icsO9IGRlxYg8L3A+\r\n" +
				"--X--\r\n",
			wantHTML: "<p>Dobrý deň</p>",
		},
		{
			name: "charset of an html part inside multipart/related",
			raw: "From: a@b.c\r\nContent-Type: multipart/related; boundary=X\r\n\r\n" +
				"--X\r\nContent-Type: text/html; charset=iso-8859-2\r\n\r\n<p>Dobr\xfd de\xf2</p>\r\n" +
				"--X--\r\n",
			wantHTML: "<p>Dobrý deň</p>",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			msg, err := ParseEmail(strings.NewReader(c.raw))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			if msg.TextBody != c.wantText || msg.HTMLBody != c.wantHTML {
				t.Errorf("text %q html %q, want text %q html %q", msg.TextBody, msg.HTMLBody, c.wantText, c.wantHTML)
			}
		})
	}
}

// A header that does not parse must not cost the sender their message.
func TestParseToleratesBadHeaders(t *testing.T) {
	raw := "From: not an address <<\r\nDate: yesterday\r\nSubject: still here\r\n" +
		"Content-Type: text/plain\r\n\r\nhi\r\n"

	msg, err := ParseEmail(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if msg.Subject != "still here" || !msg.Date.IsZero() || len(msg.From) != 0 {
		t.Errorf("subject %q date %v from %v", msg.Subject, msg.Date, msg.From)
	}
}

// Mutually nested multipart/alternative and multipart/related must stop at
// maxMultipartDepth instead of recursing for as long as the message lasts.
func TestParseEmailRejectsDeepNesting(t *testing.T) {
	nested := func(depth int) string {
		var b strings.Builder

		b.WriteString("From: a@b.c\r\nContent-Type: multipart/alternative; boundary=x0\r\n\r\n")

		for i := 0; i < depth; i++ {
			kind := "related"
			if i%2 == 1 {
				kind = "alternative"
			}

			fmt.Fprintf(&b, "--x%d\r\nContent-Type: multipart/%s; boundary=x%d\r\n\r\n", i, kind, i+1)
		}

		return b.String()
	}

	// The generated message is never terminated, so shallow nesting still ends
	// in an EOF -- what matters is that it is not the depth error.
	if _, err := ParseEmail(strings.NewReader(nested(3))); errors.Is(err, errMultipartTooDeep) {
		t.Errorf("3 levels: %v", err)
	}

	if _, err := ParseEmail(strings.NewReader(nested(20000))); !errors.Is(err, errMultipartTooDeep) {
		t.Errorf("20000 levels: err = %v, want errMultipartTooDeep", err)
	}
}
