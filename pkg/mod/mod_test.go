// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package mod

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/efficientgo/core/testutil"
	"golang.org/x/mod/module"
)

func expectContent(t *testing.T, expected string, file string) {
	t.Helper()

	b, err := os.ReadFile(file)
	testutil.Ok(t, err)
	testutil.Equals(t, expected, string(b))
}

func TestFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	t.Run("open empty", func(t *testing.T) {
		t.Parallel()

		testFile := filepath.Join(tmpDir, "test.mod")

		_, err := OpenFile(testFile)
		testutil.NotOk(t, err)
		testutil.Equals(t, "open "+testFile+": no such file or directory", err.Error())

		testutil.Ok(t, os.WriteFile(testFile, []byte(``), os.ModePerm))
		mf, err := OpenFile(testFile)
		testutil.Ok(t, err)

		testutil.Equals(t, testFile, mf.Filepath())
		p, comment := mf.Module()
		testutil.Equals(t, "", p)
		testutil.Equals(t, "", comment)
		testutil.Equals(t, []string(nil), mf.Comments())
		testutil.Equals(t, "", mf.GoVersion())
		testutil.Equals(t, 0, len(mf.RequireDirectives()))
		testutil.Equals(t, 0, len(mf.ReplaceDirectives()))
		testutil.Equals(t, 0, len(mf.ExcludeDirectives()))
		testutil.Equals(t, 0, len(mf.RetractDirectives()))
	})
	t.Run("open mod file & modify.", func(t *testing.T) {
		t.Parallel()

		testFile := filepath.Join(tmpDir, "test2.mod")

		testutil.Ok(t, os.WriteFile(testFile, []byte(`module github.com/bwplotka/bingo

go 1.17

// Comment 1.

require (
	github.com/prometheus/prometheus v2.4.3+incompatible // cmd/prometheus yolo
	github.com/Masterminds/semver v1.5.0
	github.com/efficientgo/core v1.0.0-rc.0
	github.com/oklog/run v1.1.0
	golang.org/x/mod v0.5.1
	mvdan.cc/sh/v3 v3.4.3
)

// Comment 2.

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20220330033206-e17cdc41300f // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
)

replace mvdan.cc/sh/v3 => mvdan.cc/sh/v3 v3.2

retract (
	// Wrongly formatted.
	v1.0.0
)

exclude mvdan.cc/sh/v3 v3.4.4

// Comment 3.


`), os.ModePerm))

		mf, err := OpenFile(testFile)
		testutil.Ok(t, err)

		p, comment := mf.Module()
		testutil.Equals(t, "github.com/bwplotka/bingo", p)
		testutil.Equals(t, "", comment)

		testutil.Equals(t, []string{"Comment 1.", "Comment 2.", "Comment 3."}, mf.Comments())
		testutil.Equals(t, "1.17", mf.GoVersion())

		reqDirectives := mf.RequireDirectives()
		testutil.Equals(t, 11, len(reqDirectives))
		testutil.Equals(t, "github.com/prometheus/prometheus", reqDirectives[0].Module.Path)
		testutil.Equals(t, "v2.4.3+incompatible", reqDirectives[0].Module.Version)
		testutil.Equals(t, "cmd/prometheus yolo", reqDirectives[0].ExtraSuffixComment)
		testutil.Equals(t, false, reqDirectives[0].Indirect)

		testutil.Equals(t, "github.com/Masterminds/semver", reqDirectives[1].Module.Path)
		testutil.Equals(t, "v1.5.0", reqDirectives[1].Module.Version)
		testutil.Equals(t, "", reqDirectives[1].ExtraSuffixComment)
		testutil.Equals(t, false, reqDirectives[1].Indirect)

		testutil.Equals(t, "github.com/davecgh/go-spew", reqDirectives[6].Module.Path)
		testutil.Equals(t, "v1.1.1", reqDirectives[6].Module.Version)
		testutil.Equals(t, "", reqDirectives[6].ExtraSuffixComment)
		testutil.Equals(t, true, reqDirectives[6].Indirect)

		replDirectives := mf.ReplaceDirectives()
		testutil.Equals(t, 1, len(mf.ReplaceDirectives()))
		testutil.Equals(t, "mvdan.cc/sh/v3", replDirectives[0].Old.String())
		testutil.Equals(t, "mvdan.cc/sh/v3@v3.2.0", replDirectives[0].New.String())

		excludeDirectives := mf.ExcludeDirectives()
		testutil.Equals(t, 1, len(excludeDirectives))
		testutil.Equals(t, "mvdan.cc/sh/v3@v3.4.4", excludeDirectives[0].Module.String())

		retractDirectives := mf.RetractDirectives()
		testutil.Equals(t, 1, len(retractDirectives))
		testutil.Equals(t, "v1.0.0", retractDirectives[0].VersionInterval.High)
		testutil.Equals(t, "v1.0.0", retractDirectives[0].VersionInterval.Low)
		testutil.Equals(t, "Wrongly formatted.", retractDirectives[0].Rationale)

		// Modify.
		testutil.Ok(t, mf.SetModule("_", "yolo"))
		testutil.Ok(t, mf.AddComment("Let's go!"))
		testutil.Ok(t, mf.SetGoVersion("1.18"))
		testutil.Ok(t, mf.SetRequireDirectives(RequireDirective{Module: module.Version{Path: "my/module", Version: "v1.0.0"}, ExtraSuffixComment: "yolo"}))
		testutil.Ok(t, mf.SetReplaceDirectives(ReplaceDirective{Old: module.Version{Path: "my/module", Version: "v1.0.0"}, New: module.Version{Path: "my/module", Version: "v1.2.2"}}))
		testutil.Ok(t, mf.SetExcludeDirectives(ExcludeDirective{Module: module.Version{Path: "my/module", Version: "v1.1.0"}}))
		testutil.Ok(t, mf.SetRetractDirectives(RetractDirective{VersionInterval: VersionInterval{High: "v1.0.0", Low: "v0.9.0"}, Rationale: "I don't know"}))

		expectContent(t, `module _ // yolo

go 1.18

// Comment 1.

// Comment 2.

require my/module v1.0.0 // yolo

// Comment 3.

// Let's go!

replace my/module v1.0.0 => my/module v1.2.2

exclude my/module v1.1.0

// I don't know
retract [v0.9.0, v1.0.0]
`, testFile)
		p, comment = mf.Module()
		testutil.Equals(t, "_", p)
		testutil.Equals(t, "yolo", comment)

		testutil.Equals(t, []string{"Comment 1.", "Comment 2.", "Comment 3.", "Let's go!", "I don't know"}, mf.Comments())
		testutil.Equals(t, "1.18", mf.GoVersion())

		reqDirectives = mf.RequireDirectives()
		testutil.Equals(t, 1, len(reqDirectives))
		testutil.Equals(t, "my/module", reqDirectives[0].Module.Path)
		testutil.Equals(t, "v1.0.0", reqDirectives[0].Module.Version)
		testutil.Equals(t, "yolo", reqDirectives[0].ExtraSuffixComment)
		testutil.Equals(t, false, reqDirectives[0].Indirect)

		replDirectives = mf.ReplaceDirectives()
		testutil.Equals(t, 1, len(mf.ReplaceDirectives()))
		testutil.Equals(t, "my/module@v1.0.0", replDirectives[0].Old.String())
		testutil.Equals(t, "my/module@v1.2.2", replDirectives[0].New.String())

		excludeDirectives = mf.ExcludeDirectives()
		testutil.Equals(t, 1, len(excludeDirectives))
		testutil.Equals(t, "my/module@v1.1.0", excludeDirectives[0].Module.String())

		retractDirectives = mf.RetractDirectives()
		testutil.Equals(t, 1, len(retractDirectives))
		testutil.Equals(t, "v1.0.0", retractDirectives[0].VersionInterval.High)
		testutil.Equals(t, "v0.9.0", retractDirectives[0].VersionInterval.Low)
		testutil.Equals(t, "I don't know", retractDirectives[0].Rationale)
	})
}
