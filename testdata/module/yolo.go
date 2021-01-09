package module

import "github.com/pkg/errors"

const Version = "2"

func Yolo() error {
	return errors.Errorf("some error")
}
