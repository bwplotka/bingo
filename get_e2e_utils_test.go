// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package main_test

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwplotka/bingo/pkg/cpy"
	"github.com/efficientgo/core/errors"
	"github.com/efficientgo/core/testutil"
	"github.com/otiai10/copy"
)

type testProject struct {
	pwd, root   string
	isGoProject bool
}

func newTestProject(t testing.TB, base string, target string, isGoProject bool) *testProject {
	t.Helper()

	wd, err := os.Getwd()
	testutil.Ok(t, err)

	// NOTE: go.1.23 support os.CopyFS. https://github.com/golang/go/issues/62484
	err = copy.Copy(base, target)
	testutil.Ok(t, err)

	if isGoProject {
		err = copy.Copy(filepath.Join(wd, "testdata", "main.go"), filepath.Join(target, "main.go"))
		testutil.Ok(t, err)
		err = copy.Copy(filepath.Join(wd, "testdata", "go.mod"), filepath.Join(target, "go.mod"))
		testutil.Ok(t, err)
		err = copy.Copy(filepath.Join(wd, "testdata", "go.sum"), filepath.Join(target, "go.sum"))
		testutil.Ok(t, err)
	}

	err = copy.Copy(filepath.Join(wd, "testdata", "Makefile"), filepath.Join(target, "Makefile"))
	testutil.Ok(t, err)
	return &testProject{
		pwd:         wd,
		root:        target,
		isGoProject: isGoProject,
	}
}

func (g *testProject) assertNotChanged(t testing.TB, except ...string) {
	t.Helper()

	if g.isGoProject {
		g.assertGoModDidNotChange(t).assertGoSumDidNotChange(t)
		except = append(except, "main.go", "go.sum", "go.mod")
	}
	g.assertProjectRootIsClean(t, except...)
}

func (g *testProject) assertGoModDidNotChange(t testing.TB) *testProject {
	t.Helper()

	a, err := os.ReadFile(filepath.Join(g.root, "go.mod"))
	testutil.Ok(t, err)

	b, err := os.ReadFile(filepath.Join(g.pwd, "testdata", "go.mod"))
	testutil.Ok(t, err)

	testutil.Equals(t, string(b), string(a))

	return g
}

func (g *testProject) assertGoSumDidNotChange(t testing.TB) *testProject {
	t.Helper()

	a, err := os.ReadFile(filepath.Join(g.root, "go.sum"))
	testutil.Ok(t, err)

	b, err := os.ReadFile(filepath.Join(g.pwd, "testdata", "go.sum"))
	testutil.Ok(t, err)

	testutil.Equals(t, string(b), string(a))
	return g
}

func (g *testProject) assertProjectRootIsClean(t testing.TB, extra ...string) *testProject {
	t.Helper()

	expected := map[string]struct{}{
		"Makefile": {},
	}
	for _, e := range extra {
		expected[e] = struct{}{}
	}
	if g.isGoProject {
		expected["go.mod"] = struct{}{}
		expected["go.sum"] = struct{}{}
		expected["main.go"] = struct{}{}
	}

	i, err := os.ReadDir(g.root)
	testutil.Ok(t, err)
	got := map[string]struct{}{}
	for _, f := range i {
		got[f.Name()] = struct{}{}
	}
	testutil.Equals(t, expected, got)

	return g
}

type goEnv struct {
	goroot, gopath, goproxy, gobin, gocache, tmpDir string
}

func execCmd(dir string, env []string, command string, args ...string) (string, error) {
	var cmd *exec.Cmd
	if env == nil {
		cmd = exec.Command(command, args...)
	} else {
		// Since we want to have synthetic PATH, do not allows unspecified paths.
		// Otherwise unit test environment PATH will be used for lookup as exec.LookPath is not parametrized.
		// TL;DR: command has to have path separator.
		cmd = &exec.Cmd{
			Env:  env,
			Path: command,
			Args: append([]string{command}, args...),
		}
	}
	cmd.Dir = dir
	var b bytes.Buffer
	cmd.Stdout = &b
	cmd.Stderr = &b
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return "", errors.Newf("error while running command %q; out: %s; err: %v", cmd.String(), b.String(), err)

		}
		return "", errors.Newf("error while running command %q; out: %s; err: %v", cmd.String(), b.String(), err)
	}
	return b.String(), nil
}

func buildInitialGobin(t *testing.T, targetDir string) {
	t.Helper()

	wd, err := os.Getwd()
	testutil.Ok(t, err)

	testutil.Ok(t, os.Setenv("GOBIN", filepath.Join(wd, ".bin")))

	_, err = execCmd(wd, nil, "make", "build")
	testutil.Ok(t, err)
	err = cpy.Executable(filepath.Join(os.Getenv("GOBIN"), bingoBin), targetDir)
	testutil.Ok(t, err)
}

func makePath(t *testing.T) string {
	t.Helper()

	makePath, err := exec.LookPath("make")
	testutil.Ok(t, err)
	return strings.TrimSuffix(makePath, "\n")
}

func newIsolatedGoEnv(t testing.TB, goproxy string) *goEnv {
	tmpDir, err := os.MkdirTemp("", "bingo-tmpgoenv")
	testutil.Ok(t, err)

	tmpDir, err = filepath.Abs(tmpDir)
	testutil.Ok(t, err)

	goRoot, err := exec.LookPath("go")
	testutil.Ok(t, err)

	gopath := filepath.Join(tmpDir, "gopath")
	return &goEnv{
		tmpDir: tmpDir,
		goroot: filepath.Dir(goRoot),
		gopath: gopath,
		// Making sure $GOBIN is actually different than standard one to test advanced stuff.
		gobin:   filepath.Join(tmpDir, "bin"),
		gocache: filepath.Join(tmpDir, "gocache"),
		goproxy: goproxy,
	}
}

// Clear clears all go env dirs but not goroot, gopath and gocache.
func (g *goEnv) Clear(t testing.TB) {
	t.Helper()

	err := ChmodRecursively(g.tmpDir, 0777)
	testutil.Ok(t, err)

	dirs, err := os.ReadDir(g.tmpDir)
	testutil.Ok(t, err)

	for _, d := range dirs {
		switch filepath.Join(g.tmpDir, d.Name()) {
		case g.gocache, g.goroot, g.gopath:
		default:
			testutil.Ok(t, os.RemoveAll(filepath.Join(g.tmpDir, d.Name())))
		}
	}
}

func (g *goEnv) TmpDir() string {
	return g.tmpDir
}

func (g *goEnv) syntheticEnv() []string {
	return []string{
		// Make sure we don't require clang to build etc.
		"CGO_ENABLED=0",
		fmt.Sprintf("PATH=%s:%s:%s", g.goroot, g.tmpDir, g.gobin),
		fmt.Sprintf("GO=%s", filepath.Join(g.goroot, "go")),
		fmt.Sprintf("GOBIN=%s", g.gobin),
		fmt.Sprintf("GOPATH=%s", g.gopath),
		fmt.Sprintf("GOCACHE=%s", g.gocache),
		fmt.Sprintf("GOPROXY=%s", g.goproxy),
	}
}

func (g *goEnv) ExecOutput(t testing.TB, dir string, command string, args ...string) string {
	t.Helper()

	b, err := execCmd(dir, g.syntheticEnv(), command, args...)
	testutil.Ok(t, err)
	return b
}

func (g *goEnv) ExpectErr(dir string, command string, args ...string) error {
	_, err := execCmd(dir, g.syntheticEnv(), command, args...)
	return err
}

func (g *goEnv) existingBinaries(t *testing.T) []string {
	t.Helper()

	var filenames []string
	files, err := os.ReadDir(g.gobin)
	if os.IsNotExist(err) {
		return []string{}
	}
	testutil.Ok(t, err)

	for _, f := range files {
		if f.IsDir() {
			t.Fatal("Did not expect directory in gobin", g.gobin, "got", f.Name())
		}
		filenames = append(filenames, f.Name())
	}
	return filenames
}

func (g *goEnv) Close(t testing.TB) {
	t.Helper()

	err := ChmodRecursively(g.tmpDir, 0777)
	testutil.Ok(t, err)
	testutil.Ok(t, os.RemoveAll(g.tmpDir))
}

func ChmodRecursively(root string, mode fs.FileMode) error {
	return filepath.Walk(root,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			err = os.Chmod(path, mode)
			if err != nil {
				return err
			}
			return nil
		})
}
