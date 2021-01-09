package module

import (
	errors "github.com/efficientgo/tools/core"
)

func Yolo() error {
	return errors.Errorf("some error")
}
