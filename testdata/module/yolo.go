// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package module

import "github.com/pkg/errors"

const Version = "2.1"

func Yolo() error {
	return errors.Errorf("some error")
}
