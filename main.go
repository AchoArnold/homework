package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-errors/errors"
	"github.com/joho/godotenv"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	bolt "go.etcd.io/bbolt"
)

type logSender struct{}

func (s logSender) Send(to, from EmailAddress, email Email) (err error) {
	log.Printf("to: %#+v\n", to)
	log.Printf("from: %#+v\n", from)
	log.Printf("email: %#+v\n", email)
	return nil
}

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

type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Error struct {
	Err          error
	Message string
	ApiErrorCode string
	isApiError   bool
}

func (err *Error) Error() string {
	return err.Message
}

func NewError(currentError error, message string) error {
	return &Error{
		Err:        currentError,
		Message:    strings.Join([]string{message, currentError.Error()}, "\n"),
		isApiError: false,
	}
}

func NewApiError(message string, errorCode string) error {
	return &Error{
		Err:          errors.New("Api Error"),
		Message:      strings.Join([]string{message, "Api Error"}, "\n"),
		ApiErrorCode: errorCode,
		isApiError:   true,
	}
}

type AuthResponse struct {
	AccessToken string    `json:"access_token"`
	Error       *ApiError `json:"error,omitempty"`
}

type TestTakersRequest struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func (request *TestTakersRequest) ToMap() map[string]string {
	return map[string]string{
		"limit":  strconv.Itoa(request.Limit),
		"offset": strconv.Itoa(request.Offset),
	}
}

type TestTakersResponse struct {
	TestTakers []TestTaker `json:"test_takers"`
	Total      int         `json:"total"`
	Error *ApiError        `json:"error,omitempty"`
}
type ContactInfo struct {
	Phone        string `json:"phone"`
	FullName     string `json:"full_name"`
	Street       string `json:"street"`
	City         string `json:"city"`
	ZipCode      string `json:"zip_code"`
	State        string `json:"state"`
	Country      string `json:"country"`
	Website      string `json:"website"`
	Linkedin     string `json:"linkedin"`
	ContactEmail string `json:"contact_email"`
}
type TestTaker struct {
	ID                    int      `json:"id"`
	Name                  string      `json:"name"`
	Email                 string      `json:"email"`
	URL                   string      `json:"url"`
	HireState             string      `json:"hire_state"`
	SubmittedInTime       bool        `json:"submitted_in_time"`
	IsDemo                bool        `json:"is_demo"`
	Percent               int         `json:"percent"`
	StartedAt             int         `json:"started_at"`
	FinishedAt            int         `json:"finished_at"`
	ContactInfo           ContactInfo `json:"contact_info"`
	TestDurationInSeconds int         `json:"test_duration_in_seconds"`
}

type ApiError struct {
	Type string `json:"type"`
}

const ContentTypeJson = "application/json"

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func getAccessToken() (*AuthResponse, error) {
	authRequest := AuthRequest{
		Email:    os.Getenv("API_AUTH_EMAIL"),
		Password: os.Getenv("API_AUTH_PASSWORD"),
	}

	request, err := createPostRequest(os.Getenv("API_AUTH_ENDPOINT"), authRequest)
	if err != nil {
		return nil, err
	}

	response, err := doHTTPRequest(request)
	if err != nil {
		return nil, err
	}

	var accessTokenResponse AuthResponse

	err = decodeJSON(&accessTokenResponse, response.Body)
	if err != nil {
		return nil, err
	}

	if accessTokenResponse.Error != nil {
		return nil, NewApiError("api returned an error when fetching the access token", accessTokenResponse.Error.Type)
	}

	return &accessTokenResponse, nil
}

func getTestTakers(accessToken *AuthResponse, limit, offset int) (*TestTakersResponse, error) {
	testTakersInput := &TestTakersRequest{limit, offset}

	request, err := createGetRequest(os.Getenv("API_TEST_TAKERS_ENDPOINT"), testTakersInput.ToMap(), accessToken)
	if err != nil {
		return nil, err
	}

	response, err := doHTTPRequest(request)
	if err != nil {
		return nil, err
	}

	var testTakersResponse TestTakersResponse
	err = decodeJSON(&testTakersResponse, response.Body)
	if err != nil {
		return nil, err
	}

	if testTakersResponse.Error != nil {
		return nil, NewApiError("api returned an error when fetching test takers", testTakersResponse.Error.Type)
	}

	return &testTakersResponse, nil
}

func decodeJSON(variable interface{}, reader io.Reader) error {
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(variable)
	if err != nil {
		return NewError(err, fmt.Sprintf("Cannot Decode Content into object %v", variable))
	}

	return nil
}

func createGetRequest(url string, params map[string]string, accessToken *AuthResponse) (*http.Request, error) {
	apiRequest, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, NewError(err, "cannot create request for URL: "+url)
	}

	query := apiRequest.URL.Query()
	for key, val := range params {
		query.Add(key, val)
	}
	apiRequest.URL.RawQuery = query.Encode()

	apiRequest.Header.Set("Accept", ContentTypeJson)
	apiRequest.Header.Set("Content-Type", ContentTypeJson)
	apiRequest.Header.Set("Authorization", "Bearer "+accessToken.AccessToken)

	return apiRequest, nil
}

func createPostRequest(url string, request interface{}) (*http.Request, error) {
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, NewError(err, fmt.Sprintf("cannot serialize request to json %v", request))
	}

	apiRequest, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(requestBytes))
	if err != nil {
		return nil, NewError(err, "cannot create request for URL: "+url)
	}

	apiRequest.Header.Set("Accept", ContentTypeJson)
	apiRequest.Header.Set("Content-Type", ContentTypeJson)

	return apiRequest, nil
}

func doHTTPRequest(request *http.Request) (*http.Response, error) {
	client := &http.Client{}

	apiResponse, err := client.Do(request)
	if err != nil {
		return nil, NewError(
			err,
			fmt.Sprintf("Cannot execute %s request for %s: ", request.Method, request.URL.String()),
		)
	}

	return apiResponse, nil
}

func emailIsValid(email string) bool {
	emailRegexp := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	return emailRegexp.MatchString(email)
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

func StoreLastFinishedAt(db *bolt.DB, timestamp int) error {
	err := db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("config"))
		if err == nil {
			return err
		}

		if bucket == nil {
			return errors.New("config bucket does not exist")
		}

		err = bucket.Put([]byte("last_finished_at"), []byte(strconv.Itoa(timestamp)))
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return NewError(err,"could not store last finished at")
	}

	return nil
}

func FetchLastFinishedAt(db *bolt.DB) (int, error) {
	var dbData []byte
	err := db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("config"))
		if bucket == nil {
			return errors.New("config bucket does not exist")
		}

		dbData = bucket.Get([]byte("last_finished_at"))
		if dbData == nil {
			return errors.New("invalid data in bucket")
		}

		return nil
	})

	if err != nil {
		return -1, nil
	}

	timestamp, err := strconv.Atoi(string(dbData))
	if err != nil {
		return -1, NewError(err, fmt.Sprintf("Could not convert bytes %s into int", string(dbData)))
	}

	return timestamp,nil
}

// Depending on how we want to handle critical errors, Errors which reach thi function are critical and affect the flow
// of the application so they should be looked into immediately.
func logError(err error, parameters ...interface{}) {
	log.Println(err.Error(), parameters)
}


func JsonEncode(itemToEncode interface{}) ([]byte, error) {
	encodedObject, err := json.Marshal(itemToEncode)
	if err != nil {
		return nil, NewError(err, fmt.Sprintf("could not marshal object to json: %v", itemToEncode))
	}

	return encodedObject, nil
}

func StoreFailedTestTakerEmails(db *bolt.DB, taker TestTakerEmail) error {
	err := db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("failed_test_taker_emails"))
		if err != nil {
			return NewError(err, "Cannot create bucket 'failed_test_taker_emails'")
		}

		testTakerEmailAsBytes, err := JsonEncode(taker)
		if err != nil {
			return NewError(err, fmt.Sprintf("could not marshal test taker email %v", taker))
		}

		err = bucket.Put([]byte(strconv.Itoa(taker.TestTakerId)), testTakerEmailAsBytes)
		if err != nil {
			return NewError(err, fmt.Sprintf("could not save failed test taker with ID %d", taker.TestTakerId))
		}

		return nil
	})

	if err != nil {
		return NewError(err, "Could not store failed test taker email")
	}

	return nil
}

func StoreTestTakerEmail(db *bolt.DB, taker TestTakerEmail) error {
	err := db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("test_taker_email"))
		if err != nil {
			return NewError(err, "Cannot create bucket 'test_taker_email'")
		}

		testTakerEmailAsBytes, err := JsonEncode(taker)
		if err != nil {
			log.Println(err.Error())
			return NewError(err, fmt.Sprintf("could not marshal test taker email %v", taker))
		}

		err = bucket.Put([]byte(strconv.Itoa(taker.TestTakerId)), testTakerEmailAsBytes)
		if err != nil {
			return NewError(err, fmt.Sprintf("could not save test taker with ID %d", taker.TestTakerId))
		}

		return nil
	})

	if err != nil {
		return NewError(err, "Could not store test taker email")
	}

	return nil
}

func FetchEmailForTestTaker(db *bolt.DB, testTaker TestTaker) (*TestTakerEmail, error) {
	var testTakerEmailAsBytes []byte

	_ = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("test_taker_email"))
		if bucket == nil {
			return nil
		}

		testTakerEmailAsBytes = bucket.Get([]byte(strconv.Itoa(testTaker.ID)))
		return nil
	})

	if testTakerEmailAsBytes == nil {
		return nil,nil
	}

	var testTakerEmail TestTakerEmail
	err := decodeJSON(&testTakerEmail, bytes.NewBuffer(testTakerEmailAsBytes))
	if err != nil {
		return nil, NewError(err, fmt.Sprintf("cannot unmartial bytes '%s' into TestTakerEmail struct", string(testTakerEmailAsBytes)))
	}

	return &testTakerEmail, nil
}

func main() {
	db, err := bolt.Open(os.Getenv("DB_PATH"), 0666, nil)
	if err != nil {
		log.Fatalf("cannot open database in %s", os.Getenv("DB_PATH"))
	}
	defer func() {
		err := db.Close()
		if err != nil {
			log.Printf("error closing database %s", err.Error())
		}
	}()

	for {
		log.Printf("Doing execution %s\n", time.Now().String())

		startTime := time.Now()

		handleTestTakers(db)

		time.Sleep( 10 * time.Second -  time.Since(startTime))
	}
}
