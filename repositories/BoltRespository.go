package repositories

import (
	"bytes"
	"fmt"
	"github.com/AchoArnold/homework/domain"
	"github.com/AchoArnold/homework/services/json"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
	"log"
	"strconv"
)

type BoltRepository struct {
	Client *bolt.DB
}

const (
	bucketConfig                = "config"
	bucketFailedTestTakerEmails = "failed_test_taker_emails"
	bucketTestTakerEmail        = "test_taker_email"

	keyLastFinishedAt = "last_finished_at"
)

func NewBoltRepository(dbPath string) (repository domain.Repository) {
	db, err := bolt.Open(dbPath, 0666, nil)
	if err != nil {
		log.Fatalf("cannot open database in %s", dbPath)
	}

	return &BoltRepository{Client: db}
}

func (repository *BoltRepository) StoreLastFinishedAt(timestamp int) (err error) {
	err = repository.Client.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(bucketConfig))
		if err == nil {
			return err
		}

		if bucket == nil {
			return errors.New("config bucket does not exist")
		}

		err = bucket.Put([]byte(keyLastFinishedAt), []byte(strconv.Itoa(timestamp)))
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return errors.Wrap(err, "could not store last finished at")
	}

	return nil
}

func (repository *BoltRepository) FetchLastFinishedAt() (timestamp int, err error) {
	var dbData []byte
	err = repository.Client.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketConfig))
		if bucket == nil {
			return errors.New("config bucket does not exist")
		}

		dbData = bucket.Get([]byte(keyLastFinishedAt))
		if dbData == nil {
			return errors.New("invalid data in bucket")
		}

		return nil
	})

	if err != nil {
		return domain.BaseTimestamp, nil
	}

	timestamp, err = strconv.Atoi(string(dbData))
	if err != nil {
		return domain.BaseTimestamp, errors.Wrapf(err, "could not convert bytes %s into int", string(dbData))
	}

	return timestamp, nil
}

func (repository *BoltRepository) StoreFailedTestTakerEmail(email domain.TestTakerEmail) (err error) {
	err = repository.Client.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(bucketFailedTestTakerEmails))
		if err != nil {
			return errors.Wrap(err, "Cannot create bucket 'failed_test_taker_emails'")
		}

		testTakerEmailAsBytes, err := json.JsonEncode(email)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("could not marshal test email email %#+v", email))
		}

		err = bucket.Put([]byte(strconv.Itoa(email.TestTakerId)), testTakerEmailAsBytes)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("could not save failed test email with ID %d", email.TestTakerId))
		}

		return nil
	})

	if err != nil {
		return errors.Wrap(err, "Could not store failed test email email")
	}

	return nil
}

func (repository *BoltRepository) StoreTestTakerEmail(email domain.TestTakerEmail) (err error) {
	err = repository.Client.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(bucketTestTakerEmail))
		if err != nil {
			return errors.Wrap(err, "Cannot create bucket 'test_taker_email'")
		}

		testTakerEmailAsBytes, err := json.JsonEncode(email)
		if err != nil {
			return errors.Wrapf(err, "could not marshal test email email %v", email)
		}

		err = bucket.Put([]byte(strconv.Itoa(email.TestTakerId)), testTakerEmailAsBytes)
		if err != nil {
			return errors.Wrapf(err, "could not save test email with ID %d", email.TestTakerId)
		}

		return nil
	})

	if err != nil {
		return errors.Wrap(err, "Could not store test email email")
	}

	return nil
}

func (repository *BoltRepository) FetchEmailForTestTaker(testTaker domain.TestTaker) (testTakerEmail *domain.TestTakerEmail, err error) {
	var testTakerEmailAsBytes []byte

	_ = repository.Client.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketTestTakerEmail))
		if bucket == nil {
			return nil
		}

		testTakerEmailAsBytes = bucket.Get([]byte(strconv.Itoa(testTaker.ID)))
		return nil
	})

	if testTakerEmailAsBytes == nil {
		return nil, nil
	}

	err = json.JsonDecode(&testTakerEmail, bytes.NewBuffer(testTakerEmailAsBytes))
	if err != nil {
		return nil, errors.Wrapf(err, "cannot decode bytes '%s' into TestTakerEmail struct", string(testTakerEmailAsBytes))
	}

	return testTakerEmail, nil
}
