// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package runner

import (
	"testing"

	"github.com/efficientgo/tools/core/pkg/testutil"
	"github.com/pkg/errors"
)

func TestIsSupportedVersion(t *testing.T) {
	for _, tcase := range []struct {
		output string
		err    error
	}{
		{output: "", err: errors.New("found unsupported go version: ; requires go1.14.x or higher")},
		{output: "go version go1.1 linux/amd64", err: errors.New("found unsupported go version: v1.1; requires go1.14.x or higher")},
		{output: "go version go1 linux/amd64", err: errors.New("found unsupported go version: v1; requires go1.14.x or higher")},
		{output: "go version go1.1.2 linux/amd64", err: errors.New("found unsupported go version: v1.1.2; requires go1.14.x or higher")},
		{output: "go version go1.12 linux/amd64", err: errors.New("found unsupported go version: v1.12; requires go1.14.x or higher")},
		{output: "go version go1.13 linux/amd64", err: errors.New("found unsupported go version: v1.13; requires go1.14.x or higher")},
		{output: "go version go1.13.2 linux/amd64", err: errors.New("found unsupported go version: v1.13.2; requires go1.14.x or higher")},
		{output: "go version go1.14 linux/amd64"},
		{output: "go version go1.14.2 linux/amd64"},
		{output: "go version go1.15 linux/amd64"},
		{output: "go version go1.15.44 linux/amd64"},
		{output: "go version go2 linux/amd64"},
		{output: "go version go2.1 linux/amd64"},
	} {
		t.Run(tcase.output, func(t *testing.T) {
			err := isSupportedVersion(tcase.output)
			if tcase.err != nil {
				testutil.NotOk(t, err)
				testutil.Equals(t, tcase.err.Error(), err.Error())
				return
			}
			testutil.Ok(t, err)
		})
	}
}
