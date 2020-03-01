package test_taker

import (
	"bytes"
	"encoding/json"
	"github.com/pkg/errors"
	"math"
	"net/http"
	"strconv"
)

const contentTypeJson = "application/json"

type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"Password"`
}

type ApiTestTakerService  struct {
	Username, Password, AuthApiEndpoint, TestTakersApiEndpoint string
}

type TestTakersResponse struct {
	TestTakers []ApiTestTaker `json:"test_takers"`
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

type ApiTestTaker struct {
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

func (testTakerService *ApiTestTakerService) GetNewTestTakers(repository Repository, errorHandler ErrorHandler) ([]*TestTaker, error) {
	accessToken, err := getAccessToken()
	if err != nil {
		errorHandler.Handle(err)
	}

	lastFinishedAt, err := repository.FetchLastFinishedAt(db)
	if err != nil {
		errorHandler.Handle(err)
	}

	var testTakers []TestTaker
	limit := 10
	allNewTestTakersFound := false
	testTakersResponse, err := getTestTakers(accessToken, limit, 0)
	if err != nil {
		logError(err)
	}

	if len(testTakersResponse.TestTakers) > 0 {
		err := repository.StoreLastFinishedAt(db, testTakersResponse.TestTakers[0].FinishedAt)
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
			errorHandler.handle(err)
		}

		for _, testTaker := range testTakersResponse.TestTakers {
			if lastFinishedAt > 0  &&  testTaker.FinishedAt < lastFinishedAt {
				allNewTestTakersFound = true
				break
			}
			testTakers = append(testTakers, testTaker)
		}
	}
	return nil,nil
}

func createGetRequest(url string, params map[string]string, accessToken *AuthResponse) (*http.Request, error) {
	apiRequest, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create request for URL: "+url)
	}

	query := apiRequest.URL.Query()
	for key, val := range params {
		query.Add(key, val)
	}
	apiRequest.URL.RawQuery = query.Encode()

	apiRequest.Header.Set("Accept", contentTypeJson)
	apiRequest.Header.Set("Content-Type", contentTypeJson)
	apiRequest.Header.Set("Authorization", "Bearer "+accessToken.AccessToken)

	return apiRequest, nil
}

func createPostRequest(url string, request interface{}) (*http.Request, error) {
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, errors.Wrapf(err,"cannot serialize request to json %#+v", request)
	}

	apiRequest, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(requestBytes))
	if err != nil {
		return nil, errors.Wrap(err, "cannot create request for URL: "+url)
	}

	apiRequest.Header.Set("Accept", contentTypeJson)
	apiRequest.Header.Set("Content-Type", contentTypeJson)

	return apiRequest, nil
}

func doHTTPRequest(request *http.Request) (*http.Response, error) {
	client := &http.Client{}

	apiResponse, err := client.Do(request)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot execute %s request for %s: ", request.Method, request.URL.String())
	}

	return apiResponse, nil
}

func (testTakerService *ApiTestTakerService) setAccessToken() (*AuthResponse, error) {
	authRequest := AuthRequest{
		Email:    testTakerService.Username,
		Password: testTakerService.Password,
	}

	request, err := createPostRequest(testTakerService.AuthApiEndpoint, authRequest)
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
		return nil, errors.Wrap(errors.New(accessTokenResponse.Error.Type), "api returned an error when fetching the access token")
	}

	return &accessTokenResponse, nil
}

func (testTakerService *ApiTestTakerService) getTestTakers(accessToken *AuthResponse, limit, offset int) (*TestTakersResponse, error) {
	testTakersInput := &TestTakersRequest{limit, offset}

	request, err := createGetRequest(testTakerService.TestTakersApiEndpoint, testTakersInput.ToMap(), accessToken)
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
		return nil, errors.Wrap(errors.New(testTakersResponse.Error.Type),"api returned an error when fetching test takers",)
	}

	return &testTakersResponse, nil
}