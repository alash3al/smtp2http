// Vendored from github.com/alash3al/go-smtpsrv (parser.go), which is itself a
// fork of github.com/DusanKasan/parsemail with character set decoding added.
//
// Both are unmaintained -- go-smtpsrv's last commit was 2022-09 and it pins
// emersion/go-smtp to v0.15, which blocked every dependency update. Copying the
// parser in is what let that pin go. It is ours now; see parser_test.go and the
// fuzz target for the coverage that comes with owning it.
//
// Licensed under the Apache License 2.0, same as this repository.

package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"

	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/text/encoding/ianaindex"
)

const contentTypeMultipartMixed = "multipart/mixed"
const contentTypeMultipartAlternative = "multipart/alternative"
const contentTypeMultipartRelated = "multipart/related"
const contentTypeTextHtml = "text/html"
const contentTypeTextPlain = "text/plain"

// Parse an email message read from io.Reader into parsemail.Email struct
func ParseEmail(r io.Reader) (email *Email, err error) {
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return
	}

	email, err = createEmailFromHeader(msg.Header)
	if err != nil {
		return
	}

	email.ContentType = msg.Header.Get("Content-Type")
	contentType, params, err := parseContentType(email.ContentType)
	if err != nil {
		return
	}

	switch contentType {
	case contentTypeMultipartMixed:
		email.TextBody, email.HTMLBody, email.Attachments, email.EmbeddedFiles, err = parseMultipartMixed(msg.Body, params["boundary"])
	case contentTypeMultipartAlternative:
		email.TextBody, email.HTMLBody, email.EmbeddedFiles, err = parseMultipartAlternative(msg.Body, params["boundary"])
	case contentTypeMultipartRelated:
		email.TextBody, email.HTMLBody, email.EmbeddedFiles, err = parseMultipartRelated(msg.Body, params["boundary"])
	case contentTypeTextPlain:
		newPart, err := decodeContent(msg.Body, msg.Header.Get("Content-Transfer-Encoding"), msg.Header.Get("Content-Type"))
		if err != nil {
			return email, err
		}

		message, _ := io.ReadAll(newPart)
		email.TextBody = strings.TrimSuffix(string(message[:]), "\n")
	case contentTypeTextHtml:
		newPart, err := decodeContent(msg.Body, msg.Header.Get("Content-Transfer-Encoding"), msg.Header.Get("Content-Type"))
		if err != nil {
			return email, err
		}

		message, err := io.ReadAll(newPart)
		if err != nil {
			return email, err
		}

		email.HTMLBody = strings.TrimSuffix(string(message[:]), "\n")
	default:
		email.Content, err = decodeContent(msg.Body, msg.Header.Get("Content-Transfer-Encoding"), msg.Header.Get("Content-Type"))
	}

	return
}

func createEmailFromHeader(header mail.Header) (email *Email, err error) {
	hp := headerParser{}

	email = &Email{}
	email.Subject = decodeMimeSentence(header.Get("Subject"))
	email.From = hp.parseAddressList(header.Get("From"))
	email.Sender = hp.parseAddress(header.Get("Sender"))
	email.ReplyTo = hp.parseAddressList(header.Get("Reply-To"))
	email.To = hp.parseAddressList(header.Get("To"))
	email.Cc = hp.parseAddressList(header.Get("Cc"))
	email.Bcc = hp.parseAddressList(header.Get("Bcc"))
	email.Date = hp.parseTime(header.Get("Date"))
	email.ResentFrom = hp.parseAddressList(header.Get("Resent-From"))
	email.ResentSender = hp.parseAddress(header.Get("Resent-Sender"))
	email.ResentTo = hp.parseAddressList(header.Get("Resent-To"))
	email.ResentCc = hp.parseAddressList(header.Get("Resent-Cc"))
	email.ResentBcc = hp.parseAddressList(header.Get("Resent-Bcc"))
	email.ResentMessageID = hp.parseMessageId(header.Get("Resent-Message-ID"))
	email.MessageID = hp.parseMessageId(header.Get("Message-ID"))
	email.InReplyTo = hp.parseMessageIdList(header.Get("In-Reply-To"))
	email.References = hp.parseMessageIdList(header.Get("References"))
	email.ResentDate = hp.parseTime(header.Get("Resent-Date"))

	//decode whole header for easier access to extra fields
	//todo: should we decode? aren't only standard fields mime encoded?
	email.Header, err = decodeHeaderMime(header)
	if err != nil {
		return
	}

	return
}

func parseContentType(contentTypeHeader string) (contentType string, params map[string]string, err error) {
	if contentTypeHeader == "" {
		contentType = contentTypeTextPlain
		return
	}

	return mime.ParseMediaType(contentTypeHeader)
}

func parseMultipartRelated(msg io.Reader, boundary string) (textBody, htmlBody string, embeddedFiles []EmbeddedFile, err error) {
	pmr := multipart.NewReader(msg, boundary)
	for {
		part, err := pmr.NextPart()

		if err == io.EOF {
			break
		} else if err != nil {
			return textBody, htmlBody, embeddedFiles, err
		}

		contentType, params, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if err != nil {
			return textBody, htmlBody, embeddedFiles, err
		}

		switch contentType {
		case contentTypeTextPlain:
			newPart, err := decodeContent(part, part.Header.Get("Content-Transfer-Encoding"), part.Header.Get("Content-Type"))
			if err != nil {
				return textBody, htmlBody, embeddedFiles, err
			}

			ppContent, err := io.ReadAll(newPart)
			if err != nil {
				return textBody, htmlBody, embeddedFiles, err
			}

			textBody += strings.TrimSuffix(string(ppContent[:]), "\n")
		case contentTypeTextHtml:
			newPart, err := decodeContent(part, part.Header.Get("Content-Transfer-Encoding"), part.Header.Get("Content-Type"))
			if err != nil {
				return textBody, htmlBody, embeddedFiles, err
			}

			ppContent, err := io.ReadAll(newPart)
			if err != nil {
				return textBody, htmlBody, embeddedFiles, err
			}

			htmlBody += strings.TrimSuffix(string(ppContent[:]), "\n")
		case contentTypeMultipartAlternative:
			tb, hb, ef, err := parseMultipartAlternative(part, params["boundary"])
			if err != nil {
				return textBody, htmlBody, embeddedFiles, err
			}

			htmlBody += hb
			textBody += tb
			embeddedFiles = append(embeddedFiles, ef...)
		default:
			if isEmbeddedFile(part) {
				ef, err := decodeEmbeddedFile(part)
				if err != nil {
					return textBody, htmlBody, embeddedFiles, err
				}

				embeddedFiles = append(embeddedFiles, ef)
			} else {
				return textBody, htmlBody, embeddedFiles, fmt.Errorf("can't process multipart/related inner mime type: %s", contentType)
			}
		}
	}

	return textBody, htmlBody, embeddedFiles, err
}

func decodeCharset(content io.Reader, contentTypeWithCharset string) io.Reader {
	_, params, err := mime.ParseMediaType(contentTypeWithCharset)
	if err != nil {
		return content
	}

	name := params["charset"]
	if name == "" {
		return content
	}

	// ianaindex resolves any registered charset name, case-insensitively.
	// The previous implementation matched a hardcoded list of two names by
	// exact case and required "; charset=" with a space, so lowercase
	// "windows-1252", ";charset=" without the space, and everything outside
	// ISO-8859-1 / Windows-1252 arrived undecoded.
	enc, err := ianaindex.MIME.Encoding(name)
	if err != nil || enc == nil {
		return content
	}

	return enc.NewDecoder().Reader(content)
}

func parseMultipartAlternative(msg io.Reader, boundary string) (textBody, htmlBody string, embeddedFiles []EmbeddedFile, err error) {
	pmr := multipart.NewReader(msg, boundary)
	for {
		part, err := pmr.NextPart()

		if err == io.EOF {
			break
		} else if err != nil {
			return textBody, htmlBody, embeddedFiles, err
		}

		contentType, params, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if err != nil {
			return textBody, htmlBody, embeddedFiles, err
		}

		switch contentType {
		case contentTypeTextPlain:
			newPart, err := decodeContent(part, part.Header.Get("Content-Transfer-Encoding"), part.Header.Get("Content-Type"))
			if err != nil {
				return textBody, htmlBody, embeddedFiles, err
			}

			ppContent, err := io.ReadAll(newPart)
			if err != nil {
				return textBody, htmlBody, embeddedFiles, err
			}

			textBody += strings.TrimSuffix(string(ppContent[:]), "\n")

		case contentTypeTextHtml:
			newPart, err := decodeContent(part, part.Header.Get("Content-Transfer-Encoding"), part.Header.Get("Content-Type"))
			if err != nil {
				return textBody, htmlBody, embeddedFiles, err
			}

			ppContent, err := io.ReadAll(newPart)
			if err != nil {
				return textBody, htmlBody, embeddedFiles, err
			}

			htmlBody += strings.TrimSuffix(string(ppContent[:]), "\n")

		case contentTypeMultipartRelated:
			tb, hb, ef, err := parseMultipartRelated(part, params["boundary"])
			if err != nil {
				return textBody, htmlBody, embeddedFiles, err
			}

			htmlBody += hb
			textBody += tb
			embeddedFiles = append(embeddedFiles, ef...)

		default:
			if isEmbeddedFile(part) {
				ef, err := decodeEmbeddedFile(part)
				if err != nil {
					return textBody, htmlBody, embeddedFiles, err
				}

				embeddedFiles = append(embeddedFiles, ef)
			} else {
				return textBody, htmlBody, embeddedFiles, fmt.Errorf("can't process multipart/alternative inner mime type: %s", contentType)
			}
		}
	}

	return textBody, htmlBody, embeddedFiles, err
}

func parseMultipartMixed(msg io.Reader, boundary string) (textBody, htmlBody string, attachments []Attachment, embeddedFiles []EmbeddedFile, err error) {
	mr := multipart.NewReader(msg, boundary)
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			return textBody, htmlBody, attachments, embeddedFiles, err
		}

		contentType, params, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if err != nil {
			return textBody, htmlBody, attachments, embeddedFiles, err
		}

		if contentType == contentTypeMultipartAlternative {
			textBody, htmlBody, embeddedFiles, err = parseMultipartAlternative(part, params["boundary"])
			if err != nil {
				return textBody, htmlBody, attachments, embeddedFiles, err
			}
		} else if contentType == contentTypeMultipartRelated {
			textBody, htmlBody, embeddedFiles, err = parseMultipartRelated(part, params["boundary"])
			if err != nil {
				return textBody, htmlBody, attachments, embeddedFiles, err
			}
		} else if contentType == contentTypeTextPlain {
			newPart, err := decodeContent(part, part.Header.Get("Content-Transfer-Encoding"), part.Header.Get("Content-Type"))
			if err != nil {
				return textBody, htmlBody, attachments, embeddedFiles, err
			}

			ppContent, err := io.ReadAll(newPart)
			if err != nil {
				return textBody, htmlBody, attachments, embeddedFiles, err
			}

			textBody += strings.TrimSuffix(string(ppContent[:]), "\n")
		} else if contentType == contentTypeTextHtml {
			newPart, err := decodeContent(part, part.Header.Get("Content-Transfer-Encoding"), part.Header.Get("Content-Type"))
			if err != nil {
				return textBody, htmlBody, attachments, embeddedFiles, err
			}

			ppContent, err := io.ReadAll(newPart)
			if err != nil {
				return textBody, htmlBody, attachments, embeddedFiles, err
			}

			htmlBody += strings.TrimSuffix(string(ppContent[:]), "\n")
		} else if isAttachment(part) {
			at, err := decodeAttachment(part)
			if err != nil {
				return textBody, htmlBody, attachments, embeddedFiles, err
			}

			attachments = append(attachments, at)
		} else {
			return textBody, htmlBody, attachments, embeddedFiles, fmt.Errorf("unknown multipart/mixed nested mime type: %s", contentType)
		}
	}

	return textBody, htmlBody, attachments, embeddedFiles, err
}

func decodeMimeSentence(s string) string {
	result := []string{}
	ss := strings.Split(s, " ")

	for _, word := range ss {
		dec := new(mime.WordDecoder)
		w, err := dec.Decode(word)
		if err != nil {
			if len(result) == 0 {
				w = word
			} else {
				w = " " + word
			}
		}

		result = append(result, w)
	}

	return strings.Join(result, "")
}

func decodeHeaderMime(header mail.Header) (mail.Header, error) {
	parsedHeader := map[string][]string{}

	for headerName, headerData := range header {

		parsedHeaderData := []string{}
		for _, headerValue := range headerData {
			parsedHeaderData = append(parsedHeaderData, decodeMimeSentence(headerValue))
		}

		parsedHeader[headerName] = parsedHeaderData
	}

	return mail.Header(parsedHeader), nil
}

func isEmbeddedFile(part *multipart.Part) bool {
	return part.Header.Get("Content-Transfer-Encoding") != ""
}

func decodeEmbeddedFile(part *multipart.Part) (ef EmbeddedFile, err error) {
	cid := decodeMimeSentence(part.Header.Get("Content-Id"))
	decoded, err := decodeContent(part, part.Header.Get("Content-Transfer-Encoding"), part.Header.Get("Content-Type"))
	if err != nil {
		return
	}

	ef.CID = strings.Trim(cid, "<>")
	ef.Data = decoded
	ef.ContentType = part.Header.Get("Content-Type")

	return
}

func isAttachment(part *multipart.Part) bool {
	return part.FileName() != ""
}

func decodeAttachment(part *multipart.Part) (at Attachment, err error) {
	filename := decodeMimeSentence(part.FileName())
	decoded, err := decodeContent(part, part.Header.Get("Content-Transfer-Encoding"), part.Header.Get("Content-Type"))
	if err != nil {
		return
	}

	at.Filename = filename
	at.Data = decoded
	at.ContentType = strings.Split(part.Header.Get("Content-Type"), ";")[0]

	return
}

func decodeContent(content io.Reader, encoding string, contentTypeWithCharset string) (io.Reader, error) {
	// RFC 2045 §6.1: the encoding name is case-insensitive. "8bit" and
	// "binary" are identity encodings like "7bit"; the previous exact-match
	// switch rejected the whole message for them, and for "Base64" or
	// "Quoted-Printable" in any other case.
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "base64":
		decoded := base64.NewDecoder(base64.StdEncoding, content)
		b, err := io.ReadAll(decoded)
		if err != nil {
			return nil, err
		}

		return decodeCharset(bytes.NewReader(b), contentTypeWithCharset), nil

	case "quoted-printable":
		decoded := quotedprintable.NewReader(content)
		b, err := io.ReadAll(decoded)
		if err != nil {
			return nil, err
		}

		return decodeCharset(bytes.NewReader(b), contentTypeWithCharset), nil

	case "7bit", "8bit", "binary":
		dd, err := io.ReadAll(content)
		if err != nil {
			return nil, err
		}

		return decodeCharset(bytes.NewReader(dd), contentTypeWithCharset), nil

	case "":
		return decodeCharset(content, contentTypeWithCharset), nil

	default:
		return nil, fmt.Errorf("unknown encoding: %s", encoding)
	}
}

// headerParser is deliberately lenient: a header that does not parse yields
// its zero value and the message is still delivered. Mail from real clients
// routinely carries a malformed address or an odd Date, and bouncing the
// whole message over a header the webhook may not even read would be worse.
//
// The upstream version carried an err field meant to abort on the first bad
// header, but assigned it through value receivers, so it was never seen.
// Lenient is therefore what every release has done; this just says so.
type headerParser struct{}

func (headerParser) parseAddress(s string) *mail.Address {
	if strings.Trim(s, " \n") == "" {
		return nil
	}

	ma, _ := mail.ParseAddress(s)

	return ma
}

func (headerParser) parseAddressList(s string) []*mail.Address {
	if strings.Trim(s, " \n") == "" {
		return nil
	}

	ma, _ := mail.ParseAddressList(s)

	return ma
}

func (headerParser) parseTime(s string) (t time.Time) {
	if s == "" {
		return
	}

	formats := []string{
		time.RFC1123Z,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		time.RFC1123Z + " (MST)",
		"Mon, 2 Jan 2006 15:04:05 -0700 (MST)",
	}

	for _, format := range formats {
		if parsed, err := time.Parse(format, s); err == nil {
			return parsed
		}
	}

	return
}

func (headerParser) parseMessageId(s string) string {
	return strings.Trim(s, "<> ")
}

func (hp headerParser) parseMessageIdList(s string) (result []string) {
	for _, p := range strings.Split(s, " ") {
		if strings.Trim(p, " \n") != "" {
			result = append(result, hp.parseMessageId(p))
		}
	}

	return
}

// Attachment with filename, content type and data (as a io.Reader)
type Attachment struct {
	Filename    string
	ContentType string
	Data        io.Reader
}

// EmbeddedFile with content id, content type and data (as a io.Reader)
type EmbeddedFile struct {
	CID         string
	ContentType string
	Data        io.Reader
}

// Email with fields for all the headers defined in RFC5322 with it's attachments and
type Email struct {
	Header mail.Header

	Subject    string
	Sender     *mail.Address
	From       []*mail.Address
	ReplyTo    []*mail.Address
	To         []*mail.Address
	Cc         []*mail.Address
	Bcc        []*mail.Address
	Date       time.Time
	MessageID  string
	InReplyTo  []string
	References []string

	ResentFrom      []*mail.Address
	ResentSender    *mail.Address
	ResentTo        []*mail.Address
	ResentDate      time.Time
	ResentCc        []*mail.Address
	ResentBcc       []*mail.Address
	ResentMessageID string

	ContentType string
	Content     io.Reader

	HTMLBody string
	TextBody string

	Attachments   []Attachment
	EmbeddedFiles []EmbeddedFile
}
