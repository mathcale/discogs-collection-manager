package mailer

import "github.com/resend/resend-go/v2"

type ResendMailer struct {
	client *resend.Client
	from   string
}

func NewResendMailer(apiKey, from string) *ResendMailer {
	return &ResendMailer{
		client: resend.NewClient(apiKey),
		from:   from,
	}
}

func (m *ResendMailer) Send(to, subject, body string) error {
	params := &resend.SendEmailRequest{
		From:    m.from,
		To:      []string{to},
		Subject: subject,
		Html:    body,
	}

	_, err := m.client.Emails.Send(params)
	return err
}
