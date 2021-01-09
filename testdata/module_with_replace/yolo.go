package module

import (
	errors "github.com/efficientgo/tools/core"
)

const Version = "1.1"

func Yolo() error {
	return errors.Errorf("some error")
}
