package mailer

import "log"

type LogSender struct{}

func NewLogSender() (sender LogSender) {
	return LogSender{}
}

func (s LogSender) Send(to, from EmailAddress, email Email) (err error) {
	log.Printf("to: %#+v\n", to)
	log.Printf("from: %#+v\n", from)
	log.Printf("email: %#+v\n", email)
	return nil
}
