package json

import (
	"encoding/json"
	"github.com/pkg/errors"
	"io"
)

func JsonEncode(itemToEncode interface{}) ([]byte, error) {
	encodedObject, err := json.Marshal(itemToEncode)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal object to json: %#+v", itemToEncode)
	}

	return encodedObject, nil
}


func JsonDecode(variable *interface{}, reader io.Reader) error {
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(variable)
	if err != nil {
		return errors.Wrapf(err, "cannot decode content into object %#+v", variable)
	}

	return nil
}