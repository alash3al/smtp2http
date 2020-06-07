package main

// EmailAddress ...
type EmailAddress struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"address,omitempty"`
}

// EmailAttachment ...
type EmailAttachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Data        string `json:"data"`
}

// EmailEmbeddedFile ...
type EmailEmbeddedFile struct {
	CID         string `json:"cid"`
	ContentType string `json:"content_type"`
	Data        string `json:"data"`
}

// EmailMessage ...
type EmailMessage struct {
	References []string `json:"references,omitempty"`
	SPFResult  string   `json:"spf,omitempty"`

	ID      string `json:"id,omitempty"`
	Date    string `json:"date,omitempty"`
	Subject string `json:"subject,omitempty"`

	ResentDate string `json:"resent_date,omitempty"`
	ResentID   string `json:"resent_id,omitempty"`

	Body struct {
		Text string `json:"text,omitempty"`
		HTML string `json:"html,omitempty"`
	} `json:"body"`

	Addresses struct {
		From      *EmailAddress   `json:"from"`
		To        []*EmailAddress `json:"to"`
		ReplyTo   []*EmailAddress `json:"reply_to,omitempty"`
		Cc        []*EmailAddress `json:"cc,omitempty"`
		Bcc       []*EmailAddress `json:"bcc,omitempty"`
		InReplyTo []string        `json:"in_reply_to,omitempty"`

		ResentFrom *EmailAddress   `json:"resent_from,omitempty"`
		ResentTo   []*EmailAddress `json:"resent_to,omitempty"`
		ResentCc   []*EmailAddress `json:"resent_cc,omitempty"`
		ResentBcc  []*EmailAddress `json:"resent_bcc,omitempty"`
	} `json:"addresses"`

	Attachments   []*EmailAttachment   `json:"attachments,omitempty"`
	EmbeddedFiles []*EmailEmbeddedFile `json:"embedded_files,omitempty"`
}
