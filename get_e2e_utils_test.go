// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package main_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/efficientgo/tools/core/pkg/testutil"
	"github.com/pkg/errors"
)

type testProject struct {
	pwd, root   string
	isGoProject bool
}

func newTestProject(t testing.TB, base string, target string, isGoProject bool) *testProject {
	t.Helper()

	wd, err := os.Getwd()
	testutil.Ok(t, err)

	_, err = execCmd(wd, nil, "cp", "-r", base, target)
	testutil.Ok(t, err)

	if isGoProject {
		_, err = execCmd(wd, nil, "cp", filepath.Join(wd, "testdata", "main.go"), target)
		testutil.Ok(t, err)
		_, err = execCmd(wd, nil, "cp", filepath.Join(wd, "testdata", "go.mod"), target)
		testutil.Ok(t, err)
		_, err = execCmd(wd, nil, "cp", filepath.Join(wd, "testdata", "go.sum"), target)
		testutil.Ok(t, err)
	}

	_, err = execCmd(wd, nil, "cp", filepath.Join(wd, "testdata", "Makefile"), target)
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

	a, err := ioutil.ReadFile(filepath.Join(g.root, "go.mod"))
	testutil.Ok(t, err)

	b, err := ioutil.ReadFile(filepath.Join(g.pwd, "testdata", "go.mod"))
	testutil.Ok(t, err)

	testutil.Equals(t, string(b), string(a))

	return g
}

func (g *testProject) assertGoSumDidNotChange(t testing.TB) *testProject {
	t.Helper()

	a, err := ioutil.ReadFile(filepath.Join(g.root, "go.sum"))
	testutil.Ok(t, err)

	b, err := ioutil.ReadFile(filepath.Join(g.pwd, "testdata", "go.sum"))
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

	i, err := ioutil.ReadDir(g.root)
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
			return "", errors.Errorf("error while running command %q; out: %s; err: %v", cmd.String(), b.String(), err)

		}
		return "", errors.Errorf("error while running command %q; out: %s; err: %v", cmd.String(), b.String(), err)
	}
	return b.String(), nil
}

func buildInitialGobin(t *testing.T, targetDir string) {
	t.Helper()

	wd, err := os.Getwd()
	testutil.Ok(t, err)

	_, err = execCmd(wd, nil, "make", "build")
	testutil.Ok(t, err)
	_, err = execCmd(wd, nil, "cp", filepath.Join(os.Getenv("GOBIN"), bingoBin), targetDir)
	testutil.Ok(t, err)
}

func makePath(t *testing.T) string {
	t.Helper()

	makePath, err := execCmd("", nil, "which", "make")
	testutil.Ok(t, err)
	return strings.TrimSuffix(makePath, "\n")
}

func newIsolatedGoEnv(t testing.TB, goproxy string) *goEnv {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "bingo-tmpgoenv")
	testutil.Ok(t, err)

	tmpDir, err = filepath.Abs(tmpDir)
	testutil.Ok(t, err)

	goRoot, err := execCmd("", nil, "which", "go")
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

	_, err := execCmd("", nil, "chmod", "-R", "777", g.tmpDir)
	testutil.Ok(t, err)

	dirs, err := ioutil.ReadDir(g.tmpDir)
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
	files, err := ioutil.ReadDir(g.gobin)
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

	_, err := execCmd("", nil, "chmod", "-R", "777", g.tmpDir)
	testutil.Ok(t, err)
	testutil.Ok(t, os.RemoveAll(g.tmpDir))
}
