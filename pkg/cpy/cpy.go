// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package cpy

import (
	"io"
	"os"

	"github.com/efficientgo/core/errcapture"
)

func File(src, dst string) (err error) {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer errcapture.Do(&err, source.Close, "close source")

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer errcapture.Do(&err, destination.Close, "close destination")

	buf := make([]byte, 1024)
	for {
		n, err := source.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		if _, err := destination.Write(buf[:n]); err != nil {
			return err
		}
	}
	return nil
}
