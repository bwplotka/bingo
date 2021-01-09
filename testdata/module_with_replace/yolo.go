package module

import (
	errors "github.com/bwplotka/bingo"
)

const Version = "2.7"

func Yolo() error {
	return errors.Errorf("some error")
}
