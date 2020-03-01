package handler

import (
	"log"
)

type LogErrorHandler struct {}

// Depending on how we want to handle critical errors, Errors which reach thi function are critical and affect the flow
// of the application logic so they should be looked into immediately. We could configure this to send an email to the
// SRE's on call
func (handler LogErrorHandler) HandleCriticalError(err error) {
	log.Panicln(err.Error())
}

func (handler LogErrorHandler) HandleError(err error) {
	log.Println(err)
}