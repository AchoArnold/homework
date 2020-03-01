package domain

type EmailAddress struct {
	Name    string
	Address string
}

type TestTakerEmail struct {
	TestTakerId int
	Email       string
}

type Email struct {
	Subject string
	Body    string
}

type TestTaker struct {
	ID         int
	Name       string
	Email      string
	IsDemo     bool
	Percent    int
	FinishedAt int
}