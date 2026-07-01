package mailer

type Mailer interface {
	Send(to, subject, body string) error
}
