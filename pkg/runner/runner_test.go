// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package runner

import (
	"testing"

	"github.com/efficientgo/core/errors"
	"github.com/efficientgo/core/merrors"
	"github.com/efficientgo/core/testutil"
)

func TestParseAndIsSupportedVersion(t *testing.T) {
	for _, tcase := range []struct {
		output string
		errs   error
	}{
		{output: "", errs: errors.New("unexpected go version output; expected 'go version go<semver> ...; found ")},
		{output: "go version go1.1 linux/amd64", errs: errors.New("found unsupported go version: 1.1.0; requires go 1.14.x or higher")},
		{output: "go version go1 linux/amd64", errs: errors.New("found unsupported go version: 1.0.0; requires go 1.14.x or higher")},
		{output: "go version go1.1.2 linux/amd64", errs: errors.New("found unsupported go version: 1.1.2; requires go 1.14.x or higher")},
		{output: "go version go1.12rc1 linux/amd64", errs: errors.New("found unsupported go version: 1.12.0; requires go 1.14.x or higher")},
		{output: "go version go1.12 linux/amd64", errs: errors.New("found unsupported go version: 1.12.0; requires go 1.14.x or higher")},
		{output: "go version go1.13 linux/amd64", errs: errors.New("found unsupported go version: 1.13.0; requires go 1.14.x or higher")},
		{output: "go version go1.13.2 linux/amd64", errs: errors.New("found unsupported go version: 1.13.2; requires go 1.14.x or higher")},
		{output: "go version go1.14 linux/amd64"},
		{output: "go version go1.14.2 linux/amd64"},
		{output: "go version go1.15 linux/amd64"},
		{output: "go version go1.15.44 linux/amd64"},
		{output: "go version go1.16beta1 linux/amd64"},
		{output: "go version go1.16rc1 linux/amd64"},
		{output: "go version go2 linux/amd64"},
		{output: "go version go2.1 linux/amd64"},
	} {
		t.Run(tcase.output, func(t *testing.T) {
			errs := merrors.New()
			v, err := parseGoVersion(tcase.output)
			if err != nil {
				errs.Add(err)
			}

			if v != nil {
				if err := isSupportedVersion(v); err != nil {
					errs.Add(err)
				}
			}
			if tcase.errs != nil {
				testutil.NotOk(t, errs.Err())
				testutil.Equals(t, tcase.errs.Error(), errs.Err().Error())
				return
			}
			testutil.Ok(t, errs.Err())
		})
	}
}
