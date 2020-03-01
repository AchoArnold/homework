package repositories

import (
	"bytes"
	bolt "go.etcd.io/bbolt"
	"log"
	"strconv"
	"github.com/pkg/errors"
	"fmt"
)

type BoltRepository struct {
	Client *bolt.DB
}

const (
	bucketLastFinishedAt = "last_finished_at"
	bucketConfig = "config"
	bucketFailedTestTakerEmails = "failed_test_taker_emails"
	bucketTestTakerEmail = "test_taker_email"
)

func NewBoltRepository(dbPath string) (repository *BoltRepository) {
	db, err := bolt.Open(dbPath, 0666, nil)
	if err != nil {
		log.Fatalf("cannot open database in %s", dbPath)
	}
	defer func() {
		err := db.Close()
		if err != nil {
			log.Printf("error closing database %s", err.Error())
		}
	}()

	return &BoltRepository{Client: db}
}
func (repository *BoltRepository) StoreLastFinishedAt(timestamp int) (err error) {
	err = repository.Client.Update(func(tx *bolt.Tx) error {
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
		return errors.Wrap(err,"could not store last finished at")
	}

	return nil
}

func (repository *BoltRepository) FetchLastFinishedAt() (timestamp int, err error) {
	var dbData []byte
	err = repository.Client.Update(func(tx *bolt.Tx) error {
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

	timestamp, err = strconv.Atoi(string(dbData))
	if err != nil {
		return -1, errors.Wrap(err, fmt.Sprintf("Could not convert bytes %s into int", string(dbData)))
	}

	return timestamp,nil
}


func (repository *BoltRepository) StoreFailedTestTakerEmails(email TestTakerEmail) (err error) {
	err := repository.Client.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("failed_test_taker_emails"))
		if err != nil {
			return errors.Wrap(err, "Cannot create bucket 'failed_test_taker_emails'")
		}

		testTakerEmailAsBytes, err := JsonEncode(email)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("could not marshal test email email %v", email))
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

func (repository *BoltRepository) StoreTestTakerEmail(email TestTakerEmail) (err error) {
	err := repository.Client.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("test_taker_email"))
		if err != nil {
			return errors.Wrap(err, "Cannot create bucket 'test_taker_email'")
		}

		testTakerEmailAsBytes, err := JsonEncode(email)
		if err != nil {
			log.Println(err.Error())
			return errors.Wrap(err, fmt.Sprintf("could not marshal test email email %v", email))
		}

		err = bucket.Put([]byte(strconv.Itoa(email.TestTakerId)), testTakerEmailAsBytes)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("could not save test email with ID %d", email.TestTakerId))
		}

		return nil
	})

	if err != nil {
		return errors.Wrap(err, "Could not store test email email")
	}

	return nil
}

func (repository *BoltRepository) FetchEmailForTestTaker(testTaker TestTaker) (testTakerEmail *TestTakerEmail, err error) {
	var testTakerEmailAsBytes []byte

	_ = repository.Client.View(func(tx *bolt.Tx) error {
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