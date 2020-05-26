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
	"time"

	"github.com/bwplotka/bingo/pkg/testutil"
	"github.com/pkg/errors"
)

const (
	bingoBin      = "bingo"
	defaultModDir = ".bingo"
)

// TODO(bwplotka): Test running versions. To do so we might want to setup small binary printing version at each commit.
func TestGet_E2E(t *testing.T) {
	g := newTmpGoEnv(t)
	defer g.Close(t)

	// We manually build bingo binary to make sure GOCACHE will not hit us.
	goBinPath := filepath.Join(g.tmpDir, bingoBin)
	buildInitialGobin(t, goBinPath)

	// Copy testproject at the beginning to temp dir.
	p := newInitialGoProject(t, "testdata/testproject", filepath.Join(g.tmpDir, "testproject"))
	p.assertProjectNotChanged(t)

	lastBuildTimes := map[string]time.Time{}
	t.Run("Get faillint v1.4.0 and pin for our module; clean module", func(t *testing.T) {
		defer p.assertProjectNotChanged(t, defaultModDir)

		fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "github.com/fatih/faillint@v1.4.0"))

		lastBuildTimes["faillint"] = g.binaryModTime(t, "faillint")

		testutil.Equals(t, "faillint: github.com/fatih/faillint@v1.4.0\n", g.ExecOutput(t, p.root, goBinPath, "list", "faillint"))
		testutil.Equals(t, "faillint: github.com/fatih/faillint@v1.4.0\n", g.ExecOutput(t, p.root, goBinPath, "list"))
	})
	t.Run("Get goimports from commit", func(t *testing.T) {
		defer p.assertProjectNotChanged(t, defaultModDir)

		fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "golang.org/x/tools/cmd/goimports@2b542361a4fc4b018c0770324a3b65d0393db1e0"))
		testutil.Equals(t, lastBuildTimes["faillint"], g.binaryModTime(t, "faillint"))

		lastBuildTimes["goimports"] = g.binaryModTime(t, "goimports")

		testutil.Equals(t, "goimports: golang.org/x/tools/cmd/goimports@v0.0.0-20200521211927-2b542361a4fc\n", g.ExecOutput(t, p.root, goBinPath, "list", "goimports"))
		testutil.Equals(t, "faillint: github.com/fatih/faillint@v1.4.0\ngoimports: golang.org/x/tools/cmd/goimports@v0.0.0-20200521211927-2b542361a4fc\n", g.ExecOutput(t, p.root, goBinPath, "list"))
	})
	t.Run("Get goimports from same commit should be noop", func(t *testing.T) {
		defer p.assertProjectNotChanged(t, defaultModDir)

		fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "golang.org/x/tools/cmd/goimports@2b542361a4fc4b018c0770324a3b65d0393db1e0"))
		testutil.Assert(t, g.binaryModTime(t, "goimports").After(lastBuildTimes["goimports"]), "goimports was not touched") // It's not noop, get reinstalls, but whatever.
		testutil.Equals(t, lastBuildTimes["faillint"], g.binaryModTime(t, "faillint"))

		lastBuildTimes["goimports"] = g.binaryModTime(t, "goimports")

		testutil.Equals(t, "faillint: github.com/fatih/faillint@v1.4.0\ngoimports: golang.org/x/tools/cmd/goimports@v0.0.0-20200521211927-2b542361a4fc\n", g.ExecOutput(t, p.root, goBinPath, "list"))
	})
	t.Run("Update goimports by path", func(t *testing.T) {
		defer p.assertProjectNotChanged(t, defaultModDir)

		fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "golang.org/x/tools/cmd/goimports@cb1345f3a375367f8439bba882e90348348288d9"))
		testutil.Assert(t, g.binaryModTime(t, "goimports").After(lastBuildTimes["goimports"]), "goimports was not touched")
		testutil.Equals(t, lastBuildTimes["faillint"], g.binaryModTime(t, "faillint"))

		lastBuildTimes["goimports"] = g.binaryModTime(t, "goimports")

		testutil.Equals(t, "faillint: github.com/fatih/faillint@v1.4.0\ngoimports: golang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\n", g.ExecOutput(t, p.root, goBinPath, "list"))
	})
	t.Run("Update faillint by name", func(t *testing.T) {
		defer p.assertProjectNotChanged(t, defaultModDir)

		fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "faillint@v1.5.0"))
		testutil.Assert(t, g.binaryModTime(t, "faillint").After(lastBuildTimes["faillint"]), "faillint was not touched")
		testutil.Equals(t, lastBuildTimes["goimports"], g.binaryModTime(t, "goimports"))

		lastBuildTimes["faillint"] = g.binaryModTime(t, "faillint")

		testutil.Equals(t, "faillint: github.com/fatih/faillint@v1.5.0\ngoimports: golang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\n", g.ExecOutput(t, p.root, goBinPath, "list"))
	})
	t.Run("Downgrade faillint by name", func(t *testing.T) {
		fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "faillint@v1.3.0"))
		testutil.Assert(t, g.binaryModTime(t, "faillint").After(lastBuildTimes["faillint"]), "faillint was not touched")
		testutil.Equals(t, lastBuildTimes["goimports"], g.binaryModTime(t, "goimports"))

		lastBuildTimes["faillint"] = g.binaryModTime(t, "faillint")

		testutil.Equals(t, "faillint: github.com/fatih/faillint@v1.3.0\ngoimports: golang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\n", g.ExecOutput(t, p.root, goBinPath, "list"))
	})
	t.Run("Get another goimports from commit, name it goimports2", func(t *testing.T) {
		defer p.assertProjectNotChanged(t, defaultModDir)

		fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "-o=goimports2", "golang.org/x/tools/cmd/goimports@7d3b6ebf133df879df3e448a8625b7029daa8954"))
		testutil.Equals(t, lastBuildTimes["faillint"], g.binaryModTime(t, "faillint"))
		testutil.Equals(t, lastBuildTimes["goimports"], g.binaryModTime(t, "goimports"))

		lastBuildTimes["goimports2"] = g.binaryModTime(t, "goimports2")

		testutil.Equals(t, "goimports2: golang.org/x/tools/cmd/goimports@v0.0.0-20200515010526-7d3b6ebf133d\n", g.ExecOutput(t, p.root, goBinPath, "list", "goimports2"))
		testutil.Equals(t, "faillint: github.com/fatih/faillint@v1.3.0\ngoimports: golang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\ngoimports2: golang.org/x/tools/cmd/goimports@v0.0.0-20200515010526-7d3b6ebf133d\n", g.ExecOutput(t, p.root, goBinPath, "list"))
	})
	t.Run("Upgrade goimports2 from commit", func(t *testing.T) {
		defer p.assertProjectNotChanged(t, defaultModDir)

		fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "goimports2@7521f6f4253398df2cb300c64dd7fba383ccdfa6"))
		testutil.Assert(t, g.binaryModTime(t, "goimports2").After(lastBuildTimes["goimports2"]), "goimports2 was not touched")
		testutil.Equals(t, lastBuildTimes["faillint"], g.binaryModTime(t, "faillint"))
		testutil.Equals(t, lastBuildTimes["goimports"], g.binaryModTime(t, "goimports"))

		lastBuildTimes["goimports2"] = g.binaryModTime(t, "goimports2")

		testutil.Equals(t, "goimports2: golang.org/x/tools/cmd/goimports@v0.0.0-20200519175826-7521f6f42533\n", g.ExecOutput(t, p.root, goBinPath, "list", "goimports2"))
		testutil.Equals(t, "faillint: github.com/fatih/faillint@v1.3.0\ngoimports: golang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\ngoimports2: golang.org/x/tools/cmd/goimports@v0.0.0-20200519175826-7521f6f42533\n", g.ExecOutput(t, p.root, goBinPath, "list"))
	})
	t.Run("Remove goimports2 by name", func(t *testing.T) {
		defer p.assertProjectNotChanged(t, defaultModDir)

		fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "goimports2@none"))
		testutil.Equals(t, lastBuildTimes["goimports2"], g.binaryModTime(t, "goimports2")) // We don't remove binaries, only mod file.
		testutil.Equals(t, lastBuildTimes["goimports"], g.binaryModTime(t, "goimports"))
		testutil.Equals(t, lastBuildTimes["faillint"], g.binaryModTime(t, "faillint"))

		testutil.Equals(t, "faillint: github.com/fatih/faillint@v1.3.0\ngoimports: golang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\n", g.ExecOutput(t, p.root, goBinPath, "list"))
	})
	t.Run("Remove goimports by path", func(t *testing.T) {
		defer p.assertProjectNotChanged(t, defaultModDir)

		fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "golang.org/x/tools/cmd/goimports@none"))
		testutil.Equals(t, lastBuildTimes["goimports2"], g.binaryModTime(t, "goimports2"))
		testutil.Equals(t, lastBuildTimes["goimports"], g.binaryModTime(t, "goimports")) // We don't remove binaries, only mod file.
		testutil.Equals(t, lastBuildTimes["faillint"], g.binaryModTime(t, "faillint"))

		testutil.Equals(t, "faillint: github.com/fatih/faillint@v1.3.0\n", g.ExecOutput(t, p.root, goBinPath, "list"))
	})
	t.Run("Remove faillint by name", func(t *testing.T) {
		defer p.assertProjectNotChanged(t, defaultModDir)

		fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "faillint@none"))
		testutil.Equals(t, lastBuildTimes["goimports2"], g.binaryModTime(t, "goimports2"))
		testutil.Equals(t, lastBuildTimes["goimports"], g.binaryModTime(t, "goimports"))
		testutil.Equals(t, lastBuildTimes["faillint"], g.binaryModTime(t, "faillint")) // We don't remove binaries, only mod file.

		testutil.Equals(t, "", g.ExecOutput(t, p.root, goBinPath, "list"))
	})
}

func TestGet_WithPreExistingGobin_E2E(t *testing.T) {
	g := newTmpGoEnv(t)
	defer g.Close(t)

	// We manually build bingo binary to make sure GOCACHE will not hit us.
	goBinPath := filepath.Join(g.tmpDir, bingoBin)
	buildInitialGobin(t, goBinPath)

	// Copy testproject at the beginning to temp dir.
	p := newInitialGoProject(t, "testdata/testproject_with_bingo", filepath.Join(g.tmpDir, "testproject2"))
	p.assertProjectNotChanged(t)

	g.assertBinaryNotInGoBin(t, "faillint")
	g.assertBinaryNotInGoBin(t, "goimports")
	g.assertBinaryNotInGoBin(t, "goimports2")
	testutil.Equals(t, "faillint: github.com/fatih/faillint@v1.3.0\ngoimports: golang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\ngoimports2: golang.org/x/tools/cmd/goimports@v0.0.0-20200519175826-7521f6f42533\n", g.ExecOutput(t, p.root, goBinPath, "list"))

	// Get all binaries by doing 'bingo get'.
	defer p.assertProjectNotChanged(t, defaultModDir)

	fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get"))

	g.assertBinaryInGoBin(t, "faillint")
	g.assertBinaryInGoBin(t, "goimports")
	g.assertBinaryInGoBin(t, "goimports2")
	testutil.Equals(t, "faillint: github.com/fatih/faillint@v1.3.0\ngoimports: golang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\ngoimports2: golang.org/x/tools/cmd/goimports@v0.0.0-20200519175826-7521f6f42533\n", g.ExecOutput(t, p.root, goBinPath, "list"))
}

func TestGet__WithPreExistingGobin_NativeGoBuild(t *testing.T) {
	g := newTmpGoEnv(t)
	defer g.Close(t)

	// Copy testproject at the beginning to temp dir.
	// NOTE: No bingo binary is required here.
	p := newInitialGoProject(t, "testdata/testproject_with_bingo", filepath.Join(g.tmpDir, "testproject2"))
	p.assertProjectNotChanged(t)

	g.assertBinaryNotInGoBin(t, "faillint")
	g.assertBinaryNotInGoBin(t, "goimports")
	g.assertBinaryNotInGoBin(t, "goimports2")

	// Get all binaries by doing native go build.
	defer p.assertProjectNotChanged(t, defaultModDir)

	_, err := execCmd(p.root, nil, "go", "build", "-modfile="+filepath.Join(defaultModDir, "goimports.mod"), "-o="+filepath.Join(g.bingo, "goimports"), "golang.org/x/tools/cmd/goimports")
	testutil.Ok(t, err)
	_, err = execCmd(p.root, nil, "go", "build", "-modfile="+filepath.Join(defaultModDir, "faillint.mod"), "-o="+filepath.Join(g.bingo, "faillint"), "github.com/fatih/faillint")
	testutil.Ok(t, err)
	_, err = execCmd(p.root, nil, "go", "build", "-modfile="+filepath.Join(defaultModDir, "goimports2.mod"), "-o="+filepath.Join(g.bingo, "goimports2"), "golang.org/x/tools/cmd/goimports")
	testutil.Ok(t, err)

	g.assertBinaryInGoBin(t, "faillint")
	g.assertBinaryInGoBin(t, "goimports")
	g.assertBinaryInGoBin(t, "goimports2")
}

func TestGetWithMakefile_E2E(t *testing.T) {
	// Make is one of test requirement.
	makePath := makePath(t)
	t.Run("-m one by one", func(t *testing.T) {
		g := newTmpGoEnv(t)
		defer g.Close(t)

		// We manually build bingo binary to make sure GOCACHE will not hit us.
		goBinPath := filepath.Join(g.tmpDir, bingoBin)
		buildInitialGobin(t, goBinPath)

		// Copy testproject at the beginning to temp dir.
		prjRoot := filepath.Join(g.tmpDir, "testproject")
		p := newInitialGoProject(t, "testdata/testproject_makefile", prjRoot)
		p.assertProjectNotChanged(t)

		testutil.NotOk(t, g.ExectErr(p.root, makePath, "faillint-exists"))
		testutil.NotOk(t, g.ExectErr(p.root, makePath, "goimports-exists"))
		testutil.NotOk(t, g.ExectErr(p.root, makePath, "goimports2-exists"))

		fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "-m", "github.com/fatih/faillint@v1.4.0"))
		fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "golang.org/x/tools/cmd/goimports@cb1345f3a375367f8439bba882e90348348288d9"))
		fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "-m", "-o=goimports2", "golang.org/x/tools/cmd/goimports@cb1345f3a375367f8439bba882e90348348288d9"))

		testutil.Equals(t, "checking faillint\n", g.ExecOutput(t, p.root, makePath, "faillint-exists"))
		testutil.Equals(t, "checking goimports\n", g.ExecOutput(t, p.root, makePath, "goimports-exists"))
		testutil.Equals(t, "checking goimports2\n", g.ExecOutput(t, p.root, makePath, "goimports2-exists"))

		t.Run("modify binary file, expect reinstall", func(t *testing.T) {
			_, err := execCmd(g.bingo, nil, "rm", "faillint")
			testutil.Ok(t, err)
			_, err = execCmd(g.bingo, nil, "touch", "faillint")
			testutil.Ok(t, err)

			testutil.Equals(t, "checking faillint\n", g.ExecOutput(t, p.root, makePath, "faillint-exists"))
			testutil.Equals(t, "checking goimports\n", g.ExecOutput(t, p.root, makePath, "goimports-exists"))
			testutil.Equals(t, "checking goimports2\n", g.ExecOutput(t, p.root, makePath, "goimports2-exists"))
		})

		t.Run("delete makefile", func(t *testing.T) {
			fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "faillint@none"))
			fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "goimports@none"))
			fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "goimports2@none"))

			_, err := os.Stat(filepath.Join(p.root, ".bingo", "Makefile.binary-variables"))
			testutil.NotOk(t, err)
		})
	})
	t.Run("-m all at the end", func(t *testing.T) {
		g := newTmpGoEnv(t)
		defer g.Close(t)

		// We manually build bingo binary to make sure GOCACHE will not hit us.
		goBinPath := filepath.Join(g.tmpDir, bingoBin)
		buildInitialGobin(t, goBinPath)

		// Copy testproject at the beginning to temp dir.
		prjRoot := filepath.Join(g.tmpDir, "testproject")
		p := newInitialGoProject(t, "testdata/testproject_makefile", prjRoot)
		p.assertProjectNotChanged(t)

		testutil.NotOk(t, g.ExectErr(p.root, makePath, "faillint-exists"))
		testutil.NotOk(t, g.ExectErr(p.root, makePath, "goimports-exists"))
		testutil.NotOk(t, g.ExectErr(p.root, makePath, "goimports2-exists"))

		fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "-m=false", "github.com/fatih/faillint@v1.4.0"))
		fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "-m=false", "golang.org/x/tools/cmd/goimports@cb1345f3a375367f8439bba882e90348348288d9"))
		fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "-m=false", "-o=goimports2", "golang.org/x/tools/cmd/goimports@cb1345f3a375367f8439bba882e90348348288d9"))

		testutil.NotOk(t, g.ExectErr(p.root, makePath, "faillint-exists"))
		testutil.NotOk(t, g.ExectErr(p.root, makePath, "goimports-exists"))
		testutil.NotOk(t, g.ExectErr(p.root, makePath, "goimports2-exists"))

		fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "-m"))
		testutil.Equals(t, "checking faillint\n", g.ExecOutput(t, p.root, makePath, "faillint-exists"))
		testutil.Equals(t, "checking goimports\n", g.ExecOutput(t, p.root, makePath, "goimports-exists"))
		testutil.Equals(t, "checking goimports2\n", g.ExecOutput(t, p.root, makePath, "goimports2-exists"))
	})
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
	wd, err := os.Getwd()
	testutil.Ok(t, err)

	_, err = execCmd(wd, nil, "make", "build")
	testutil.Ok(t, err)
	_, err = execCmd(wd, nil, "cp", filepath.Join(os.Getenv("GOBIN"), bingoBin), targetDir)
	testutil.Ok(t, err)
}

func makePath(t *testing.T) string {
	makePath, err := execCmd("", nil, "which", "make")
	testutil.Ok(t, err)
	return strings.TrimSuffix(makePath, "\n")
}

type goProject struct {
	pwd, root, testproject string
}

func newInitialGoProject(t testing.TB, testproject string, dir string) *goProject {
	wd, err := os.Getwd()
	testutil.Ok(t, err)

	_, err = execCmd(wd, nil, "cp", "-r", testproject, dir)
	testutil.Ok(t, err)
	return &goProject{
		pwd:         wd,
		root:        dir,
		testproject: testproject,
	}
}
func (g *goProject) assertProjectNotChanged(t testing.TB, except ...string) {
	g.assertGoModDidNotChange(t).assertGoSumDidNotChange(t).assertProjectRootIsClean(t, except...)
}

func (g *goProject) assertGoModDidNotChange(t testing.TB) *goProject {
	a, err := ioutil.ReadFile(filepath.Join(g.root, "go.mod"))
	testutil.Ok(t, err)

	b, err := ioutil.ReadFile(filepath.Join(g.pwd, filepath.Join(g.testproject, "go.mod")))
	testutil.Ok(t, err)

	testutil.Equals(t, string(b), string(a))

	return g
}

func (g *goProject) assertGoSumDidNotChange(t testing.TB) *goProject {
	a, err := ioutil.ReadFile(filepath.Join(g.root, "go.sum"))
	testutil.Ok(t, err)

	b, err := ioutil.ReadFile(filepath.Join(g.pwd, filepath.Join(g.testproject, "go.sum")))
	testutil.Ok(t, err)

	testutil.Equals(t, string(b), string(a))
	return g
}

func (g *goProject) assertProjectRootIsClean(t testing.TB, extra ...string) *goProject {
	expected := map[string]struct{}{}
	for _, e := range extra {
		expected[e] = struct{}{}
	}
	i, err := ioutil.ReadDir(filepath.Join(g.pwd, g.testproject))
	testutil.Ok(t, err)
	for _, f := range i {
		expected[f.Name()] = struct{}{}
	}

	i, err = ioutil.ReadDir(g.root)
	testutil.Ok(t, err)
	got := map[string]struct{}{}
	for _, f := range i {
		got[f.Name()] = struct{}{}
	}
	testutil.Equals(t, expected, got)

	return g
}

type goEnv struct {
	goroot, gopath, bingo, gocache, tmpDir string
}

func newTmpGoEnv(t testing.TB) *goEnv {
	tmpDir, err := ioutil.TempDir(".", "bingo-tmpgoenv")
	testutil.Ok(t, err)

	tmpDir, err = filepath.Abs(tmpDir)
	testutil.Ok(t, err)

	goRoot, err := execCmd("", nil, "which", "go")
	testutil.Ok(t, err)

	gopath := filepath.Join(tmpDir, "gopath")
	return &goEnv{
		tmpDir:  tmpDir,
		goroot:  filepath.Dir(goRoot),
		gopath:  gopath,
		bingo:   filepath.Join(gopath, "bin"),
		gocache: filepath.Join(tmpDir, "gocache"),
	}
}

func (g *goEnv) TmpDir() string {
	return g.tmpDir
}

func (g *goEnv) syntheticEnv() []string {
	return []string{
		fmt.Sprintf("PATH=%s:%s:%s", g.goroot, g.tmpDir, g.bingo),
		fmt.Sprintf("GOBIN=%s", g.bingo),
		fmt.Sprintf("GOPATH=%s", g.gopath),
		fmt.Sprintf("GOCACHE=%s", g.gocache),
	}
}

func (g *goEnv) ExecOutput(t testing.TB, dir string, command string, args ...string) string {
	b, err := execCmd(dir, g.syntheticEnv(), command, args...)
	testutil.Ok(t, err)
	return b
}

func (g *goEnv) ExectErr(dir string, command string, args ...string) error {
	_, err := execCmd(dir, g.syntheticEnv(), command, args...)
	return err
}

func (g *goEnv) binaryModTime(t testing.TB, bin string) time.Time {
	i, err := os.Stat(filepath.Join(g.bingo, bin))
	testutil.Ok(t, err)
	return i.ModTime()
}

func (g *goEnv) assertBinaryNotInGoBin(t testing.TB, bin string) {
	_, err := os.Stat(filepath.Join(g.bingo, bin))
	testutil.Assert(t, os.IsNotExist(err), "binary exists? %v", err)
}

func (g *goEnv) assertBinaryInGoBin(t testing.TB, bin string) {
	g.binaryModTime(t, bin)
}

func (g *goEnv) Close(t testing.TB) {
	_, err := execCmd("", nil, "chmod", "-R", "777", g.tmpDir)
	testutil.Ok(t, err)
	testutil.Ok(t, os.RemoveAll(g.tmpDir))
}
