package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
	"strconv"
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
	Message      string
	ApiErrorCode string
	isApiError   bool
}

func (err *Error) Error() string {
	return err.Message
}

func NewError(currentError error, message string) *Error {
	return &Error{
		Err:        currentError,
		Message:    message,
		isApiError: false,
	}
}

func NewApiError(message string, errorCode string) *Error {
	return &Error{
		Err:          nil,
		Message:      message,
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
	TestTakers []TestTakers `json:"test_takers"`
	Total      int          `json:"total"`
	Error *ApiError `json:"error,omitempty"`
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
type TestTakers struct {
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

func getAccessToken() (*AuthResponse, *Error) {
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

func getTestTakers(accessToken *AuthResponse, limit, offset int) (*TestTakersResponse, *Error) {
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

func decodeJSON(variable interface{}, reader io.Reader) *Error {
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(variable)
	if err != nil {
		log.Printf("%s", err.Error())
		return NewError(err, fmt.Sprintf("Cannot Decode Content into object %v", variable))
	}

	return nil
}

func createGetRequest(url string, params map[string]string, accessToken *AuthResponse) (*http.Request, *Error) {
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

func createPostRequest(url string, request interface{}) (*http.Request, *Error) {
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

func doHTTPRequest(request *http.Request) (*http.Response, *Error) {
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
	var value []byte
	err := db.Update(func(tx *bolt.Tx) error {
		bucket,err := tx.CreateBucketIfNotExists([]byte("config"))
		if err != nil {
			return err
		}

		value = bucket.Get([]byte("last_finished_at"))
		return nil
	})
	if err != nil {
		log.Printf("%v", err)
	}

	println("were're here 1")

	lastFinishedAt := 0
	if value != nil {
		lastFinishedAt, err = strconv.Atoi(string(value))
		if err != nil {
			log.Println("Error converting byte to integer", )
		}
	}

	println("were're here 2")

	accessToken, errors := getAccessToken()
	if errors != nil {
		log.Fatalf("Error %v", err)
	}

	var testTakers []TestTakers
	for {
		limit := 10
		allNewTestTakarsFound := false
		testTakersResponse, err := getTestTakers(accessToken, limit, 0)
		if err != nil {
			log.Fatalf("Error %v", err)
		}

		if testTakersResponse.Total <= 10 {
			break
		}

		for _, testTaker := range testTakersResponse.TestTakers {
			if lastFinishedAt > 0  &&  testTaker.FinishedAt < lastFinishedAt {
				allNewTestTakarsFound = true
				break
			}
			testTakers = append(testTakers, testTaker)
		}

		if allNewTestTakarsFound || testTakersResponse.Total < limit {
			break
		}

		for i := 1; float64(i) < math.Ceil(float64(testTakersResponse.Total) / float64(limit)); i++ {
			testTakersResponse, err := getTestTakers(accessToken, limit, limit * i)
			if err != nil {
				log.Fatalf("Error %v", err)
			}

			for _, testTaker := range testTakersResponse.TestTakers {
				if lastFinishedAt > 0  &&  testTaker.FinishedAt < lastFinishedAt {
					allNewTestTakarsFound = true
					break
				}
				testTakers = append(testTakers, testTaker)
			}

			if allNewTestTakarsFound {
				break
			}

		}

		break

	}

	log.Printf("Total Test Takers = %d", len(testTakers))

	var email logSender
	for _, testTaker := range testTakers {
		if testTaker.Percent >= 80 && !testTaker.IsDemo && emailIsValid(testTaker.ContactInfo.ContactEmail) {
			err := email.Send(
				EmailAddress{Name: os.Getenv("MAIL_FROM"), Address: os.Getenv("MAIL_FROM_EMAIL")},
				EmailAddress{Name: testTaker.ContactInfo.FullName, Address: testTaker.ContactInfo.ContactEmail},
				Email{
					Subject: os.Getenv("MAIL_SUBJECT"),
					Body:    os.Getenv("MAIL_BODY"),
				},
			)

			if err != nil {
				log.Println("Error sending email")
			}

			err = Store(db, "emails_sent", strconv.Itoa(testTaker.ID), "true")
			if err != nil {
				log.Printf("%v", err)
			}
		}
	}


}

func Store(db *bolt.DB, bucket string, key string, value string) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}

		err = bucket.Put([]byte(key), []byte(value))
		if err != nil {
			return err
		}

		return nil
	})
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
