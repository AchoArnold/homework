package domain

type Sender interface {
	Send(to, from EmailAddress, email Email) error
}

type TestTakerService interface {
	GetNewTestTakers() (newTestTakers []*TestTaker, err error)
}

type Repository interface {
	StoreLastFinishedAt(timestamp int) (err error)
	FetchLastFinishedAt() (timestamp int, err error)
	StoreFailedTestTakerEmail(email TestTakerEmail) (err error)
	StoreTestTakerEmail(email TestTakerEmail) (err error)
	FetchEmailForTestTaker(testTaker TestTaker) (testTakerEmail *TestTakerEmail, err error)
}


type ErrorHandler interface {
	HandleError(err error)
}
