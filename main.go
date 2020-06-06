package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/alash3al/go-smtpsrv/v3"
	"github.com/go-resty/resty/v2"
)

func main() {
	cfg := smtpsrv.ServerConfig{
		ListenAddr:      *flagListenAddr,
		MaxMessageBytes: int(*flagMaxMessageSize),
		BannerDomain:    *flagServerName,
		Handler: smtpsrv.HandlerFunc(func(c *smtpsrv.Context) error {
			msg, err := c.Parse()
			if err != nil {
				return errors.New("Cannot read your message: " + err.Error())
			}

			spfResult, _, _ := c.SPF()

			req := resty.New().R()

			// set the url-encoded-data
			req.SetFormData(map[string]string{
				"id":                     msg.MessageID,
				"subject":                msg.Subject,
				"body[text]":             string(msg.TextBody),
				"body[html]":             string(msg.HTMLBody),
				"addresses[from]":        c.From().Address,
				"addresses[to]":          strings.Join(extractEmails(msg.To), ","),
				"addresses[in-reply-to]": strings.Join(msg.InReplyTo, ","),
				"addresses[cc]":          strings.Join(extractEmails(msg.Cc), ","),
				"addresses[bcc]":         strings.Join(extractEmails(msg.Bcc), ","),
				"spf_result":             strings.ToLower(spfResult.String()),
			})

			// set the files "attachments"
			for i, file := range msg.Attachments {
				iStr := strconv.Itoa(i)
				req.SetFileReader("file["+iStr+"]", file.Filename, (file.Data))
			}

			// submit the form
			resp, err := req.Post(*flagWebhook)
			if err != nil {
				return errors.New("E1: Cannot accept your message due to internal error, please report that to our engineers")
			} else if resp.StatusCode() != 200 {
				return errors.New("E2: Cannot accept your message due to internal error, please report that to our engineers")
			}

			return nil
		}),
	}

	fmt.Println(smtpsrv.ListenAndServe(&cfg))
}
