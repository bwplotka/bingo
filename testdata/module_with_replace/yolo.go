package module

import (
	errors "github.com/efficientgo/tools/copyright"
)

const Version = "2.8"

func Yolo() error {
	return errors.Errorf("some error")
}
