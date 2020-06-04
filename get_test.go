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

	"github.com/bwplotka/bingo/pkg/testutil"
	"github.com/bwplotka/bingo/pkg/version"
	"github.com/pkg/errors"
)

const (
	bingoBin      = "bingo"
	defaultModDir = ".bingo"
)

// TODO(bwplotka): Test running versions. To do so we might want to setup small binary printing Version at each commit.
// TODO(bwplotka): Add test cases for array versions.
// TODO(bwplotka): Test renames.
func TestGet(t *testing.T) {
	currTestCaseDir := fmt.Sprintf("testdata/testproject_with_bingo_%s", strings.ReplaceAll(version.Version, ".", "_"))

	t.Run("Empty project", func(t *testing.T) {
		for _, isGoProject := range []bool{false, true} {
			t.Run(fmt.Sprintf("isGoProject=%v", isGoProject), func(t *testing.T) {
				g := newTmpGoEnv(t)
				defer g.Close(t)

				// We manually build bingo binary to make sure GOCACHE will not hit us.
				goBinPath := filepath.Join(g.tmpDir, bingoBin)
				buildInitialGobin(t, goBinPath)

				testutil.Ok(t, os.MkdirAll(filepath.Join(g.tmpDir, "newproject"), os.ModePerm))
				p := newTestProject(t, filepath.Join(g.tmpDir, "newproject"), filepath.Join(g.tmpDir, "testproject"), isGoProject)
				p.assertNotChanged(t)

				testutil.Assert(t, !g.binaryExists("faillint-v1.3.0"), "binary exists")
				testutil.Assert(t, !g.binaryExists("faillint-v1.4.0"), "binary exists")
				testutil.Assert(t, !g.binaryExists("faillint-v1.5.0"), "binary exists")

				testutil.Assert(t, !g.binaryExists("goimports-v0.0.0-20200522201501-cb1345f3a375"), "binary exists")
				testutil.Assert(t, !g.binaryExists("goimports-v0.0.0-20200521211927-2b542361a4fc"), "binary exists")

				testutil.Assert(t, !g.binaryExists("goimports2-v0.0.0-20200522201501-cb1345f3a375"), "binary exists")
				testutil.Assert(t, !g.binaryExists("goimports2-v0.0.0-20200519175826-7521f6f42533"), "binary exists")

				t.Run("Get faillint v1.4.0 and pin for our module; clean module", func(t *testing.T) {
					defer p.assertNotChanged(t, defaultModDir)

					fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "github.com/fatih/faillint@v1.4.0"))
					testutil.Equals(t, "faillint<faillint-v1.4.0>: github.com/fatih/faillint@v1.4.0\n", g.ExecOutput(t, p.root, goBinPath, "list", "faillint"))
					testutil.Equals(t, "faillint<faillint-v1.4.0>: github.com/fatih/faillint@v1.4.0\n", g.ExecOutput(t, p.root, goBinPath, "list"))

					testutil.Assert(t, g.binaryExists("faillint-v1.4.0"), "binary does not exists")
				})
				t.Run("Get goimports from commit", func(t *testing.T) {
					defer p.assertNotChanged(t, defaultModDir)

					fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "golang.org/x/tools/cmd/goimports@2b542361a4fc4b018c0770324a3b65d0393db1e0"))
					testutil.Equals(t, "goimports<goimports-v0.0.0-20200521211927-2b542361a4fc>: golang.org/x/tools/cmd/goimports@v0.0.0-20200521211927-2b542361a4fc\n", g.ExecOutput(t, p.root, goBinPath, "list", "goimports"))
					testutil.Equals(t, "faillint<faillint-v1.4.0>: github.com/fatih/faillint@v1.4.0\ngoimports<goimports-v0.0.0-20200521211927-2b542361a4fc>: golang.org/x/tools/cmd/goimports@v0.0.0-20200521211927-2b542361a4fc\n", g.ExecOutput(t, p.root, goBinPath, "list"))

					testutil.Assert(t, g.binaryExists("faillint-v1.4.0"), "binary does not exists")
					testutil.Assert(t, g.binaryExists("goimports-v0.0.0-20200521211927-2b542361a4fc"), "binary does not exists")

				})
				t.Run("Get goimports from same commit should be noop", func(t *testing.T) {
					defer p.assertNotChanged(t, defaultModDir)

					// TODO(bwplotka): Assert if actually noop.
					fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "golang.org/x/tools/cmd/goimports@2b542361a4fc4b018c0770324a3b65d0393db1e0"))
					testutil.Equals(t, "faillint<faillint-v1.4.0>: github.com/fatih/faillint@v1.4.0\ngoimports<goimports-v0.0.0-20200521211927-2b542361a4fc>: golang.org/x/tools/cmd/goimports@v0.0.0-20200521211927-2b542361a4fc\n", g.ExecOutput(t, p.root, goBinPath, "list"))

					testutil.Assert(t, g.binaryExists("faillint-v1.4.0"), "binary does not exists")
					testutil.Assert(t, g.binaryExists("goimports-v0.0.0-20200521211927-2b542361a4fc"), "binary does not exists")
				})
				t.Run("Update goimports by path", func(t *testing.T) {
					defer p.assertNotChanged(t, defaultModDir)

					fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "golang.org/x/tools/cmd/goimports@cb1345f3a375367f8439bba882e90348348288d9"))
					testutil.Equals(t, "faillint<faillint-v1.4.0>: github.com/fatih/faillint@v1.4.0\ngoimports<goimports-v0.0.0-20200522201501-cb1345f3a375>: golang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\n", g.ExecOutput(t, p.root, goBinPath, "list"))

					testutil.Assert(t, g.binaryExists("faillint-v1.4.0"), "binary does not exists")
					testutil.Assert(t, g.binaryExists("goimports-v0.0.0-20200522201501-cb1345f3a375"), "binary does not exists")
				})
				t.Run("Update faillint by name", func(t *testing.T) {
					defer p.assertNotChanged(t, defaultModDir)

					fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "faillint@v1.5.0"))
					testutil.Equals(t, "faillint<faillint-v1.5.0>: github.com/fatih/faillint@v1.5.0\ngoimports<goimports-v0.0.0-20200522201501-cb1345f3a375>: golang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\n", g.ExecOutput(t, p.root, goBinPath, "list"))

					testutil.Assert(t, g.binaryExists("faillint-v1.5.0"), "binary does not exists")
					testutil.Assert(t, g.binaryExists("goimports-v0.0.0-20200522201501-cb1345f3a375"), "binary does not exists")
				})
				t.Run("Downgrade faillint by name", func(t *testing.T) {
					fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "faillint@v1.3.0"))
					testutil.Equals(t, "faillint<faillint-v1.3.0>: github.com/fatih/faillint@v1.3.0\ngoimports<goimports-v0.0.0-20200522201501-cb1345f3a375>: golang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\n", g.ExecOutput(t, p.root, goBinPath, "list"))

					testutil.Assert(t, g.binaryExists("faillint-v1.3.0"), "binary does not exists")
					testutil.Assert(t, g.binaryExists("goimports-v0.0.0-20200522201501-cb1345f3a375"), "binary does not exists")
				})
				t.Run("Get another goimports from commit, name it goimports2", func(t *testing.T) {
					defer p.assertNotChanged(t, defaultModDir)

					fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "-n=goimports2", "golang.org/x/tools/cmd/goimports@7d3b6ebf133df879df3e448a8625b7029daa8954"))
					testutil.Equals(t, "goimports2<goimports2-v0.0.0-20200515010526-7d3b6ebf133d>: golang.org/x/tools/cmd/goimports@v0.0.0-20200515010526-7d3b6ebf133d\n", g.ExecOutput(t, p.root, goBinPath, "list", "goimports2"))
					testutil.Equals(t, "faillint<faillint-v1.3.0>: github.com/fatih/faillint@v1.3.0\ngoimports<goimports-v0.0.0-20200522201501-cb1345f3a375>: golang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\ngoimports2<goimports2-v0.0.0-20200515010526-7d3b6ebf133d>: golang.org/x/tools/cmd/goimports@v0.0.0-20200515010526-7d3b6ebf133d\n", g.ExecOutput(t, p.root, goBinPath, "list"))

					testutil.Assert(t, g.binaryExists("faillint-v1.3.0"), "binary does not exists")
					testutil.Assert(t, g.binaryExists("goimports-v0.0.0-20200522201501-cb1345f3a375"), "binary does not exists")
					testutil.Assert(t, g.binaryExists("goimports2-v0.0.0-20200515010526-7d3b6ebf133d"), "binary does not exists")
				})
				t.Run("Upgrade goimports2 from commit", func(t *testing.T) {
					defer p.assertNotChanged(t, defaultModDir)

					fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "goimports2@7521f6f4253398df2cb300c64dd7fba383ccdfa6"))
					testutil.Equals(t, "faillint<faillint-v1.3.0>: github.com/fatih/faillint@v1.3.0\ngoimports<goimports-v0.0.0-20200522201501-cb1345f3a375>: golang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\ngoimports2<goimports2-v0.0.0-20200519175826-7521f6f42533>: golang.org/x/tools/cmd/goimports@v0.0.0-20200519175826-7521f6f42533\n", g.ExecOutput(t, p.root, goBinPath, "list"))

					testutil.Assert(t, g.binaryExists("faillint-v1.3.0"), "binary does not exists")
					testutil.Assert(t, g.binaryExists("goimports-v0.0.0-20200522201501-cb1345f3a375"), "binary does not exists")
					testutil.Assert(t, g.binaryExists("goimports2-v0.0.0-20200519175826-7521f6f42533"), "binary does not exists")
				})
				t.Run("Install package with go name should fail.", func(t *testing.T) {
					defer p.assertNotChanged(t, defaultModDir)

					testutil.NotOk(t, g.ExectErr(p.root, goBinPath, "get", "github.com/something/go"))
				})

				if !isGoProject {
					// Generate current Version test case for further tests. This should be committed as well if changed.
					testutil.Ok(t, os.RemoveAll(currTestCaseDir))
					testutil.Ok(t, os.MkdirAll(filepath.Join(currTestCaseDir, ".bingo"), os.ModePerm))
					_, err := execCmd("", nil, "cp", "-r", filepath.Join(p.root, ".bingo"), currTestCaseDir)
					testutil.Ok(t, err)
				}

				t.Run("Remove goimports2 by name", func(t *testing.T) {
					defer p.assertNotChanged(t, defaultModDir)

					fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "goimports2@none"))
					testutil.Equals(t, "faillint<faillint-v1.3.0>: github.com/fatih/faillint@v1.3.0\ngoimports<goimports-v0.0.0-20200522201501-cb1345f3a375>: golang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\n", g.ExecOutput(t, p.root, goBinPath, "list"))

					// We don't remove binaries.
					testutil.Assert(t, g.binaryExists("faillint-v1.3.0"), "binary does not exists")
					testutil.Assert(t, g.binaryExists("goimports-v0.0.0-20200522201501-cb1345f3a375"), "binary does not exists")
					testutil.Assert(t, g.binaryExists("goimports2-v0.0.0-20200519175826-7521f6f42533"), "binary does not exists")
				})
				t.Run("Remove goimports by path", func(t *testing.T) {
					defer p.assertNotChanged(t, defaultModDir)

					fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "golang.org/x/tools/cmd/goimports@none"))
					testutil.Equals(t, "faillint<faillint-v1.3.0>: github.com/fatih/faillint@v1.3.0\n", g.ExecOutput(t, p.root, goBinPath, "list"))

					// We don't remove binaries.
					testutil.Assert(t, g.binaryExists("faillint-v1.3.0"), "binary does not exists")
					testutil.Assert(t, g.binaryExists("goimports-v0.0.0-20200522201501-cb1345f3a375"), "binary does not exists")
					testutil.Assert(t, g.binaryExists("goimports2-v0.0.0-20200519175826-7521f6f42533"), "binary does not exists")
				})
				t.Run("Remove faillint by name", func(t *testing.T) {
					defer p.assertNotChanged(t, defaultModDir)

					fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "faillint@none"))
					testutil.Equals(t, "", g.ExecOutput(t, p.root, goBinPath, "list"))

					// We don't remove binaries.
					testutil.Assert(t, g.binaryExists("faillint-v1.3.0"), "binary does not exists")
					testutil.Assert(t, g.binaryExists("goimports-v0.0.0-20200522201501-cb1345f3a375"), "binary does not exists")
					testutil.Assert(t, g.binaryExists("goimports2-v0.0.0-20200519175826-7521f6f42533"), "binary does not exists")
				})
			})
		}
	})
	t.Run("Compatibility test", func(t *testing.T) {
		dirs, err := filepath.Glob("testdata/testproject*")
		testutil.Ok(t, err)
		for _, dir := range dirs {
			t.Run(dir, func(t *testing.T) {
				for _, isGoProject := range []bool{false, true} {
					t.Run(fmt.Sprintf("isGoProject=%v", isGoProject), func(t *testing.T) {
						t.Run("Via bingo", func(t *testing.T) {
							g := newTmpGoEnv(t)
							defer g.Close(t)

							// We manually build bingo binary to make sure GOCACHE will not hit us.
							goBinPath := filepath.Join(g.tmpDir, bingoBin)
							buildInitialGobin(t, goBinPath)

							// Copy testproject at the beginning to temp dir.
							p := newTestProject(t, dir, filepath.Join(g.tmpDir, "testproject1"), isGoProject)
							p.assertNotChanged(t, defaultModDir)

							testutil.Assert(t, !g.binaryExists("faillint-v1.3.0"), "binary exists")
							testutil.Assert(t, !g.binaryExists("goimports-v0.0.0-20200522201501-cb1345f3a375"), "binary exists")
							testutil.Assert(t, !g.binaryExists("goimports2-v0.0.0-20200519175826-7521f6f42533"), "binary exists")
							testutil.Equals(t, "faillint<faillint-v1.3.0>: github.com/fatih/faillint@v1.3.0\ngoimports<goimports-v0.0.0-20200522201501-cb1345f3a375>: golang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\ngoimports2<goimports2-v0.0.0-20200519175826-7521f6f42533>: golang.org/x/tools/cmd/goimports@v0.0.0-20200519175826-7521f6f42533\n", g.ExecOutput(t, p.root, goBinPath, "list"))

							defer p.assertNotChanged(t, defaultModDir)

							// Get all binaries by doing 'bingo get'.
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get"))

							testutil.Assert(t, g.binaryExists("faillint-v1.3.0"), "binary does not exists")
							testutil.Assert(t, g.binaryExists("goimports-v0.0.0-20200522201501-cb1345f3a375"), "binary does not exists")
							testutil.Assert(t, g.binaryExists("goimports2-v0.0.0-20200519175826-7521f6f42533"), "binary does not exists")
							testutil.Equals(t, "faillint<faillint-v1.3.0>: github.com/fatih/faillint@v1.3.0\ngoimports<goimports-v0.0.0-20200522201501-cb1345f3a375>: golang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\ngoimports2<goimports2-v0.0.0-20200519175826-7521f6f42533>: golang.org/x/tools/cmd/goimports@v0.0.0-20200519175826-7521f6f42533\n", g.ExecOutput(t, p.root, goBinPath, "list"))
						})
						t.Run("Via go", func(t *testing.T) {
							g := newTmpGoEnv(t)
							defer g.Close(t)

							// Copy testproject at the beginning to temp dir.
							// NOTE: No bingo binary is required here.
							p := newTestProject(t, dir, filepath.Join(g.tmpDir, "testproject2"), isGoProject)
							p.assertNotChanged(t, defaultModDir)

							testutil.Assert(t, !g.binaryExists("faillint-v1.3.0"), "binary exists")
							testutil.Assert(t, !g.binaryExists("goimports-v0.0.0-20200522201501-cb1345f3a375"), "binary exists")
							testutil.Assert(t, !g.binaryExists("goimports2-v0.0.0-20200519175826-7521f6f42533"), "binary exists")

							// Get all binaries by doing native go build.
							defer p.assertNotChanged(t, defaultModDir)

							if isGoProject {
								_, err := execCmd(p.root, nil, "go", "build", "-modfile="+filepath.Join(defaultModDir, "goimports.mod"), "-o="+filepath.Join(g.gobin, "goimports-v0.0.0-20200522201501-cb1345f3a375"), "golang.org/x/tools/cmd/goimports")
								testutil.Ok(t, err)
								_, err = execCmd(p.root, nil, "go", "build", "-modfile="+filepath.Join(defaultModDir, "faillint.mod"), "-o="+filepath.Join(g.gobin, "faillint-v1.3.0"), "github.com/fatih/faillint")
								testutil.Ok(t, err)
								_, err = execCmd(p.root, nil, "go", "build", "-modfile="+filepath.Join(defaultModDir, "goimports2.mod"), "-o="+filepath.Join(g.gobin, "goimports2-v0.0.0-20200519175826-7521f6f42533"), "golang.org/x/tools/cmd/goimports")
								testutil.Ok(t, err)
							}
							_, err := execCmd(filepath.Join(p.root, defaultModDir), nil, "go", "build", "-modfile=goimports.mod", "-o="+filepath.Join(g.gobin, "goimports-v0.0.0-20200522201501-cb1345f3a375"), "golang.org/x/tools/cmd/goimports")
							testutil.Ok(t, err)
							_, err = execCmd(filepath.Join(p.root, defaultModDir), nil, "go", "build", "-modfile=faillint.mod", "-o="+filepath.Join(g.gobin, "faillint-v1.3.0"), "github.com/fatih/faillint")
							testutil.Ok(t, err)
							_, err = execCmd(filepath.Join(p.root, defaultModDir), nil, "go", "build", "-modfile=goimports2.mod", "-o="+filepath.Join(g.gobin, "goimports2-v0.0.0-20200519175826-7521f6f42533"), "golang.org/x/tools/cmd/goimports")
							testutil.Ok(t, err)

							testutil.Assert(t, g.binaryExists("faillint-v1.3.0"), "binary does not exists")
							testutil.Assert(t, g.binaryExists("goimports-v0.0.0-20200522201501-cb1345f3a375"), "binary does not exists")
							testutil.Assert(t, g.binaryExists("goimports2-v0.0.0-20200519175826-7521f6f42533"), "binary does not exists")
						})
						// TODO(bwplotka): Test variables.env as well.
						t.Run("Makefile", func(t *testing.T) {
							// Make is one of test requirement.
							makePath := makePath(t)

							g := newTmpGoEnv(t)
							defer g.Close(t)

							// We manually build bingo binary to make sure GOCACHE will not hit us.
							goBinPath := filepath.Join(g.tmpDir, bingoBin)
							buildInitialGobin(t, goBinPath)

							// Copy testproject at the beginning to temp dir.
							prjRoot := filepath.Join(g.tmpDir, "testproject")
							p := newTestProject(t, dir, prjRoot, isGoProject)
							p.assertNotChanged(t, defaultModDir)

							testutil.Equals(t, "(re)installing "+g.gobin+"/faillint-v1.3.0\ngo: downloading github.com/fatih/faillint v1.3.0\ngo: downloading golang.org/x/tools v0.0.0-20200207224406-61798d64f025\nchecking faillint\n", g.ExecOutput(t, p.root, makePath, "faillint-exists"))
							testutil.Equals(t, "(re)installing "+g.gobin+"/goimports-v0.0.0-20200522201501-cb1345f3a375\ngo: downloading golang.org/x/tools v0.0.0-20200522201501-cb1345f3a375\ngo: downloading golang.org/x/mod v0.2.0\ngo: downloading golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543\nchecking goimports\n", g.ExecOutput(t, p.root, makePath, "goimports-exists"))
							testutil.Equals(t, "(re)installing "+g.gobin+"/goimports2-v0.0.0-20200519175826-7521f6f42533\ngo: downloading golang.org/x/tools v0.0.0-20200519175826-7521f6f42533\nchecking goimports2\n", g.ExecOutput(t, p.root, makePath, "goimports2-exists"))

							testutil.Equals(t, "checking faillint\n", g.ExecOutput(t, p.root, makePath, "faillint-exists"))
							testutil.Equals(t, "checking goimports\n", g.ExecOutput(t, p.root, makePath, "goimports-exists"))
							testutil.Equals(t, "checking goimports2\n", g.ExecOutput(t, p.root, makePath, "goimports2-exists"))

							t.Run("Delete binary file, expect reinstall", func(t *testing.T) {
								_, err := execCmd(g.gobin, nil, "rm", "faillint-v1.3.0")
								testutil.Ok(t, err)

								testutil.Equals(t, "(re)installing "+g.gobin+"/faillint-v1.3.0\nchecking faillint\n", g.ExecOutput(t, p.root, makePath, "faillint-exists"))
								testutil.Equals(t, "checking faillint\n", g.ExecOutput(t, p.root, makePath, "faillint-exists"))
								testutil.Equals(t, "checking goimports\n", g.ExecOutput(t, p.root, makePath, "goimports-exists"))
								testutil.Equals(t, "checking goimports2\n", g.ExecOutput(t, p.root, makePath, "goimports2-exists"))
							})
							t.Run("Delete makefile", func(t *testing.T) {
								fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "faillint@none"))
								fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "goimports@none"))
								fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "goimports2@none"))

								_, err := os.Stat(filepath.Join(p.root, ".bingo", "Variables.mk"))
								testutil.NotOk(t, err)
							})
						})
					})
				}
			})
		}
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

type testProject struct {
	pwd, root   string
	isGoProject bool
}

func newTestProject(t testing.TB, base string, target string, isGoProject bool) *testProject {
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
	if g.isGoProject {
		g.assertGoModDidNotChange(t).assertGoSumDidNotChange(t)
		except = append(except, "main.go", "go.sum", "go.mod")
	}
	g.assertProjectRootIsClean(t, except...)
}

func (g *testProject) assertGoModDidNotChange(t testing.TB) *testProject {
	a, err := ioutil.ReadFile(filepath.Join(g.root, "go.mod"))
	testutil.Ok(t, err)

	b, err := ioutil.ReadFile(filepath.Join(g.pwd, "testdata", "go.mod"))
	testutil.Ok(t, err)

	testutil.Equals(t, string(b), string(a))

	return g
}

func (g *testProject) assertGoSumDidNotChange(t testing.TB) *testProject {
	a, err := ioutil.ReadFile(filepath.Join(g.root, "go.sum"))
	testutil.Ok(t, err)

	b, err := ioutil.ReadFile(filepath.Join(g.pwd, "testdata", "go.sum"))
	testutil.Ok(t, err)

	testutil.Equals(t, string(b), string(a))
	return g
}

func (g *testProject) assertProjectRootIsClean(t testing.TB, extra ...string) *testProject {
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
	goroot, gopath, gobin, gocache, tmpDir string
}

func newTmpGoEnv(t testing.TB) *goEnv {
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
	}
}

func (g *goEnv) TmpDir() string {
	return g.tmpDir
}

func (g *goEnv) syntheticEnv() []string {
	return []string{
		fmt.Sprintf("PATH=%s:%s:%s", g.goroot, g.tmpDir, g.gobin),
		fmt.Sprintf("GO=%s", filepath.Join(g.goroot, "go")),
		fmt.Sprintf("GOBIN=%s", g.gobin),
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

func (g *goEnv) binaryExists(bin string) bool {
	_, err := os.Stat(filepath.Join(g.gobin, bin))
	return err == nil
}

func (g *goEnv) Close(t testing.TB) {
	_, err := execCmd("", nil, "chmod", "-R", "777", g.tmpDir)
	testutil.Ok(t, err)
	testutil.Ok(t, os.RemoveAll(g.tmpDir))
}
