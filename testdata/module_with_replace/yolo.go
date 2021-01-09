package module

import (
	errors "github.com/efficientgo/tools/copyright"
)

const Version = "2.4"

func Yolo() error {
	return errors.Errorf("some error")
}
