package domain

import (
	"github.com/pkg/errors"
	"github.com/AchoArnold/homework/services/validator"
	"log"
	"time"
)

type DependencyContainer struct {
	Repository       Repository
	Sender           Sender
	TestTakerService TestTakerService
	ErrorHandler     ErrorHandler
	FromEmailAddress EmailAddress
	ThanksEmail      Email
	FetchTestTakersInterval time.Duration
}


func RunApplication(container *DependencyContainer) {
	for {
		log.Printf("Doing execution %s\n", time.Now().String())
		startTime := time.Now()

		newTestTakers := fetchNewTestTakers(container)

		sendMailToEligibleTestTakers(container, newTestTakers)

		sleepDuration := container.FetchTestTakersInterval - time.Since(startTime)

		log.Printf("Sleping for %f minutes", sleepDuration.Minutes())

		time.Sleep(sleepDuration)
	}
}

func fetchNewTestTakers(container *DependencyContainer) (testTakers []TestTaker) {
	testTakers, err := container.TestTakerService.GetNewTestTakers(container.Repository, container.ErrorHandler)
	if err!= nil {
		container.ErrorHandler.HandleCriticalError(err)
	}

	return testTakers
}

func emailShouldBeSentToTestTaker(testTaker TestTaker) bool {
	return testTaker.Percent >= 80 && !testTaker.IsDemo && validator.EmailIsValid(testTaker.Email)
}

func sendMailToEligibleTestTakers(container *DependencyContainer, testTakers []TestTaker) {
	for _, testTaker := range testTakers {
		if emailShouldBeSentToTestTaker(testTaker) {
			testTakerEmail, err := container.Repository.FetchEmailForTestTaker(testTaker)
			if err != nil {
				container.ErrorHandler.HandleCriticalError(err)
				continue
			}

			if testTakerEmail != nil {
				continue
			}

			testTakerEmail = &TestTakerEmail{TestTakerId: testTaker.ID, Email: testTaker.Email}

			err = container.Sender.Send(
				container.FromEmailAddress,
				EmailAddress{Name: testTaker.Name, Address: testTaker.Email},
				container.ThanksEmail,
			)

			if err != nil {
				err := container.Repository.StoreFailedTestTakerEmail(*testTakerEmail)
				if err != nil {
					container.ErrorHandler.HandleCriticalError(errors.Wrap(err, "could not store failed test taker email"))
				}
			} else {
				err = container.Repository.StoreTestTakerEmail(*testTakerEmail)
				if err != nil {
					container.ErrorHandler.HandleCriticalError(err)
				}
			}
		}
	}
}
