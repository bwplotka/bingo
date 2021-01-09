package module

import "github.com/pkg/errors"

func Yolo() error {
	return errors.Errorf("some error")
}
