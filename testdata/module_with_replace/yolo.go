package module

import (
	errors "golang.org/x/crypto/openpgp/errors"
)

const Version = "2.6"

func Yolo() error {
	return errors.Errorf("some error")
}
