package mailer

import (
	"log"
	"github.com/AchoArnold/homework/domain"
)

type LogSender struct{}

func NewLogSender() (sender LogSender) {
	return LogSender{}
}

func (s LogSender) Send(to, from domain.EmailAddress, email domain.Email) (err error) {
	log.Printf("to: %#+v\n", to)
	log.Printf("from: %#+v\n", from)
	log.Printf("email: %#+v\n", email)
	return nil
}
