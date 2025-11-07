// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package version

import "github.com/Masterminds/semver"

// Version returns 'bingo' version.
const Version = "v0.10"

var (
	Go114 = semver.MustParse("1.14")
	Go116 = semver.MustParse("1.16")
	Go121 = semver.MustParse("1.21")
	Go124 = semver.MustParse("1.24")
	Go125 = semver.MustParse("1.25")
)
