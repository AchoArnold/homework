package main

import (
	"github.com/AchoArnold/homework/domain"
	"github.com/AchoArnold/homework/repositories"
	"github.com/AchoArnold/homework/services/mailer"
	"github.com/AchoArnold/homework/services/handler"
	"github.com/AchoArnold/homework/services/testtaker"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	domain.RunApplication(makeDependencyContainer())
}

// Simplistic Dependency Resolution container.
// Here, we use the .env file to chose which implementation should be used for which interface
func makeDependencyContainer() (dependencyContainer *domain.DependencyContainer) {
	dbPath, err := filepath.Abs(os.Getenv("DB_PATH"))
	if err != nil {
		log.Fatalf(errors.Wrap(err, "could not initialize db path").Error())
	}

	interval,err := strconv.Atoi(os.Getenv("FETCH_TEST_TAKERS_INTERVAL"))
	if err  != nil {
		log.Fatalf(errors.Wrap(err, "could not get the interval to fetch test takers").Error())
	}

	fetchTestTakersInterval := time.Minute * time.Duration(interval)

	return &domain.DependencyContainer{
		Repository:       repositories.NewBoltRepository(dbPath),
		Sender:           mailer.NewLogSender(),
		TestTakerService: &testtaker.ApiTestTakerService{
			Email:                 os.Getenv("API_AUTH_EMAIL"),
			Password:              os.Getenv("API_AUTH_PASSWORD"),
			AuthApiEndpoint:       os.Getenv("API_AUTH_ENDPOINT"),
			TestTakersApiEndpoint: os.Getenv("API_TEST_TAKERS_ENDPOINT"),
		},
		FromEmailAddress: domain.EmailAddress{
			Name:    os.Getenv("MAIL_FROM_NAME"),
			Address: os.Getenv("MAIL_FROM_EMAIL"),
		},
		ThanksEmail: domain.Email{
			Subject: os.Getenv("MAIL_SUBJECT"),
			Body:    os.Getenv("MAIL_BODY"),
		},
		ErrorHandler: &handler.LogErrorHandler{},
		FetchTestTakersInterval: fetchTestTakersInterval,
	}
}
