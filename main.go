package main

import (
	"github.com/joho/godotenv"
	"log"
	"math"
	"os"
	"time"
	bolt "go.etcd.io/bbolt"
)

type EmailAddress struct {
	Name    string
	Address string
}

type TestTakerEmail struct {
	TestTakerId int
	Email string
}

type Email struct {
	Subject string
	Body    string
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func handleTestTakers(db *bolt.DB) {
	accessToken, err := getAccessToken()
	if err != nil {
		log.Fatalf("Error %v", err)
	}

	lastFinishedAt, err := FetchLastFinishedAt(db)
	if err != nil {
		logError(err)
	}

	var testTakers []TestTaker
	limit := 10
	allNewTestTakersFound := false
	testTakersResponse, err := getTestTakers(accessToken, limit, 0)
	if err != nil {
		logError(err)
	}

	if len(testTakersResponse.TestTakers) > 0 {
		err := StoreLastFinishedAt(db, testTakersResponse.TestTakers[0].FinishedAt)
		if err != nil {
			logError(err)
		}
	}

	for _, testTaker := range testTakersResponse.TestTakers {
		if lastFinishedAt > -1  &&  testTaker.FinishedAt < lastFinishedAt {
			allNewTestTakersFound = true
			break
		}
		testTakers = append(testTakers, testTaker)
	}

	for i := 1; (float64(i) < math.Ceil(float64(testTakersResponse.Total) / float64(limit))) || allNewTestTakersFound; i++ {
		testTakersResponse, err := getTestTakers(accessToken, limit, limit * i)
		if err != nil {
			log.Fatalf("Error %v", err)
		}

		for _, testTaker := range testTakersResponse.TestTakers {
			if lastFinishedAt > 0  &&  testTaker.FinishedAt < lastFinishedAt {
				allNewTestTakersFound = true
				break
			}
			testTakers = append(testTakers, testTaker)
		}
	}

	log.Printf("Total Test Takers = %d", len(testTakers))

	var email logSender
	for _, testTaker := range testTakers {
		if testTaker.Percent >= 80 && !testTaker.IsDemo && emailIsValid(testTaker.ContactInfo.ContactEmail) {
			testTakerEmail, err := FetchEmailForTestTaker(db, testTaker)
			if err != nil {
				logError(err)
				continue
			}

			if testTakerEmail != nil {
				continue
			}

			testTakerEmail = &TestTakerEmail{
				TestTakerId: testTaker.ID,
				Email:       testTaker.Email,
			}

			err = email.Send(
				EmailAddress{Name: os.Getenv("MAIL_FROM"), Address: os.Getenv("MAIL_FROM_EMAIL")},
				EmailAddress{Name: testTaker.ContactInfo.FullName, Address: testTaker.ContactInfo.ContactEmail},
				Email{
					Subject: os.Getenv("MAIL_SUBJECT"),
					Body:    os.Getenv("MAIL_BODY"),
				},
			)

			if err != nil {
				err := StoreFailedTestTakerEmails(db, *testTakerEmail)
				if err != nil {
					logError(err, "could not send test taker email")
				}
			} else {
				err = StoreTestTakerEmail(db, *testTakerEmail)
				if err != nil {
					logError(err)
				}
			}
		}
	}
}

func main() {

	for {
		log.Printf("Doing execution %s\n", time.Now().String())

		startTime := time.Now()

		handleTestTakers(db)

		time.Sleep( 10 * time.Second -  time.Since(startTime))
	}
}
