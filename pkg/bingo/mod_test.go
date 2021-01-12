// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package bingo

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/bwplotka/bingo/pkg/runner"
	"github.com/efficientgo/tools/core/pkg/testutil"
	"golang.org/x/mod/module"
)

func TestModFile(t *testing.T) {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "bingo-mod")
	testutil.Ok(t, err)
	t.Cleanup(func() { testutil.Ok(t, os.RemoveAll(tmpDir)) })

	r, err := runner.NewRunner(context.TODO(), false, "go")
	testutil.Ok(t, err)

	t.Run("create new and close should create empty mod file with basic autogenerated meta", func(t *testing.T) {
		f, err := CreateFromExistingOrNew(context.TODO(), r, log.New(os.Stderr, "", 0), "non_existing.mod", "test.mod")
		testutil.Ok(t, err)
		testutil.Ok(t, f.Close())

		expectContent(t, `module _ // Auto generated by https://github.com/bwplotka/bingo. DO NOT EDIT

go 1.15
`, "test.mod")
	})
	t.Run("create new and close should work and produce same output", func(t *testing.T) {
		f, err := CreateFromExistingOrNew(context.TODO(), r, log.New(os.Stderr, "", 0), "test.mod", "test2.mod")
		testutil.Ok(t, err)
		testutil.Ok(t, f.Close())
		expectContent(t, `module _ // Auto generated by https://github.com/bwplotka/bingo. DO NOT EDIT

go 1.15
`, "test.mod")
		expectContent(t, `module _ // Auto generated by https://github.com/bwplotka/bingo. DO NOT EDIT

go 1.15
`, "test2.mod")
	})
	t.Run("create new and set direct require should work", func(t *testing.T) {
		f, err := CreateFromExistingOrNew(context.TODO(), r, log.New(os.Stderr, "", 0), "", "test3.mod")
		testutil.Ok(t, err)
		testutil.Ok(t, f.SetDirectRequire(Package{Module: module.Version{Path: "github.com/yolo/best/v100", Version: "v100.0.0"}, RelPath: "thebest"}))
		testutil.Equals(t, Package{Module: module.Version{Path: "github.com/yolo/best/v100", Version: "v100.0.0"}, RelPath: "thebest"}, *f.DirectPackage())
		testutil.Ok(t, f.Close())
		expectContent(t, `module _ // Auto generated by https://github.com/bwplotka/bingo. DO NOT EDIT

go 1.15

require github.com/yolo/best/v100 v100.0.0 // thebest
`, "test3.mod")
	})
	t.Run("create new and set direct require2 should work", func(t *testing.T) {
		f, err := CreateFromExistingOrNew(context.TODO(), r, log.New(os.Stderr, "", 0), "", "test4.mod")
		testutil.Ok(t, err)
		testutil.Ok(t, f.SetDirectRequire(Package{Module: module.Version{Path: "github.com/yolo/best/v100", Version: "v100.0.0"}}))
		testutil.Equals(t, Package{Module: module.Version{Path: "github.com/yolo/best/v100", Version: "v100.0.0"}}, *f.DirectPackage())
		testutil.Ok(t, f.Close())
		expectContent(t, `module _ // Auto generated by https://github.com/bwplotka/bingo. DO NOT EDIT

go 1.15

require github.com/yolo/best/v100 v100.0.0
`, "test4.mod")
	})
	t.Run("copy and set direct require to something else", func(t *testing.T) {
		f, err := CreateFromExistingOrNew(context.TODO(), r, log.New(os.Stderr, "", 0), "test3.mod", "test5.mod")
		testutil.Ok(t, err)
		testutil.Equals(t, Package{Module: module.Version{Path: "github.com/yolo/best/v100", Version: "v100.0.0"}, RelPath: "thebest"}, *f.DirectPackage())
		testutil.Ok(t, f.Flush())
		expectContent(t, `module _ // Auto generated by https://github.com/bwplotka/bingo. DO NOT EDIT

go 1.15

require github.com/yolo/best/v100 v100.0.0 // thebest
`, "test5.mod")

		testutil.Ok(t, f.SetDirectRequire(Package{Module: module.Version{Path: "github.com/yolo/not-best", Version: "v1"}}))
		testutil.Equals(t, Package{Module: module.Version{Path: "github.com/yolo/not-best", Version: "v1"}}, *f.DirectPackage())
		testutil.Ok(t, f.Close())
		expectContent(t, `module _ // Auto generated by https://github.com/bwplotka/bingo. DO NOT EDIT

go 1.15

require github.com/yolo/not-best v1
`, "test5.mod")
	})
}

func expectContent(t *testing.T, expected string, file string) {
	t.Helper()

	b, err := ioutil.ReadFile(file)
	testutil.Ok(t, err)
	testutil.Equals(t, expected, string(b))
}
