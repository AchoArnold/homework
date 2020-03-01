package domain

type Sender interface {
	Send(to, from EmailAddress, email Email) error
}

type TestTakerService interface {
	GetNewTestTakers(repository Repository, errorHandler ErrorHandler) (testTakers []TestTaker, err error)
}

type Repository interface {
	StoreLastFinishedAt(timestamp int) (err error)
	FetchLastFinishedAt() (timestamp int, err error)
	StoreFailedTestTakerEmail(email TestTakerEmail) (err error)
	StoreTestTakerEmail(email TestTakerEmail) (err error)
	FetchEmailForTestTaker(testTaker TestTaker) (testTakerEmail *TestTakerEmail, err error)
}

type ErrorHandler interface {
	HandleCriticalError(err error)
	HandleError(err error)
}

type Validator interface {
	EmailIsValid(email string) bool
}