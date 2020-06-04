package main

import (
	"errors"
	"strconv"
	"strings"

	"github.com/DusanKasan/parsemail"

	"github.com/alash3al/go-smtpsrv"
	"github.com/go-resty/resty"
	"github.com/zaccone/spf"
)

func handler(req *smtpsrv.Request) error {
	// validate the from data
	if *flagStrictValidation {
		if req.SPFResult != spf.Pass {
			return errors.New("Your host isn't configured correctly or you are a spammer -_-")
		} else if !req.Mailable {
			return errors.New("Your mail isn't valid because it cannot receive emails -_-")
		}
	}

	msg, err := parsemail.Parse(req.Message)
	if err != nil {
		return errors.New("Cannot read your message, it may be because of it exceeded the limits")
	}

	rq := resty.R()

	// set the url-encoded-data
	rq.SetFormData(map[string]string{
		"id":              msg.Header.Get("Message-ID"),
		"in-reply-to":     msg.ReplyTo,
		"subject":         msg.Subject,
		"body[text]":      string(msg.TextBody),
		"body[html]":      string(msg.HTMLBody),
		"addresses[from]": req.From,
		"addresses[to]":   strings.Join(extractEmails(msg.To), ","),
		"addresses[cc]":   strings.Join(extractEmails(msg.Cc), ","),
		"addresses[bcc]":  strings.Join(extractEmails(msg.Bcc), ","),
	})

	// set the files "attachments"
	for i, file := range msg.Attachments {
		is := strconv.Itoa(i)
		rq.SetFileReader("file["+is+"]", file.Filename, (file.Data))
	}

	// submit the form
	resp, err := rq.Post(*flagWebhook)
	if err != nil {
		return errors.New("Cannot accept your message due to internal error, please report that to our engineers, '" + (err.Error()) + "'")
	} else if resp.StatusCode() != 200 {
		return errors.New("BACKEND: " + resp.Status())
	}

	return nil
}
