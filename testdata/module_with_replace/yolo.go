package module

import (
	errors "github.com/efficientgo/tools/core/pkg/runutil"
)

const Version = "2.5"

func Yolo() error {
	return errors.Errorf("some error")
}
