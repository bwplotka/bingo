package module

import "github.com/pkg/errors"

const Version = "1.1"

func Yolo() error {
	return errors.Errorf("some error")
}
