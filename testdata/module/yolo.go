package module

import "github.com/pkg/errors"

func Yolo2() error {
	return errors.Errorf("some error")
}
