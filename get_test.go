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

				for _, tcase := range []struct {
					name string
					do   func(t *testing.T)

					existingBinaries []string
				}{
					{
						name: "Get faillint v1.4.0 and pin for our module; clean module",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "github.com/fatih/faillint@v1.4.0"))
							testutil.Equals(t, "Name\t\tBinary Name\t\tPackage @ Version\t\n----\t\t-----------\t\t-----------------\t\nfaillint\tfaillint-v1.4.0\tgithub.com/fatih/faillint@v1.4.0", g.ExecOutput(t, p.root, goBinPath, "list", "faillint"))
							testutil.Equals(t, g.ExecOutput(t, p.root, goBinPath, "list", "faillint"), g.ExecOutput(t, p.root, goBinPath, "list"))
						},
						existingBinaries: []string{"faillint-v1.4.0"},
					},
					{
						name: "Get goimports from commit",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "golang.org/x/tools/cmd/goimports@2b542361a4fc4b018c0770324a3b65d0393db1e0"))
							testutil.Equals(t, "Name\t\tBinary Name\t\t\t\t\t\t\tPackage @ Version\t\t\t\t\n----\t\t-----------\t\t\t\t\t\t\t-----------------\t\t\t\t\nfaillint\tfaillint-v1.4.0\t\t\t\t\t\tgithub.com/fatih/faillint@v1.4.0\t\ngoimports\tgoimports-v0.0.0-20200521211927-2b542361a4fc\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200521211927-2b542361a4fc", g.ExecOutput(t, p.root, goBinPath, "list"))
						},
						existingBinaries: []string{"faillint-v1.4.0", "goimports-v0.0.0-20200521211927-2b542361a4fc"},
					},
					{
						name: "Get goimports from same commit should be noop",
						do: func(t *testing.T) {
							// TODO(bwplotka): Assert if actually noop.
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "golang.org/x/tools/cmd/goimports@2b542361a4fc4b018c0770324a3b65d0393db1e0"))
						},
						existingBinaries: []string{"faillint-v1.4.0", "goimports-v0.0.0-20200521211927-2b542361a4fc"},
					},
					{
						name: "Update goimports by path",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "golang.org/x/tools/cmd/goimports@cb1345f3a375367f8439bba882e90348348288d9"))
							testutil.Equals(t, "Name\t\tBinary Name\t\t\t\t\t\t\tPackage @ Version\t\t\t\t\n----\t\t-----------\t\t\t\t\t\t\t-----------------\t\t\t\t\nfaillint\tfaillint-v1.4.0\t\t\t\t\t\tgithub.com/fatih/faillint@v1.4.0\t\ngoimports\tgoimports-v0.0.0-20200522201501-cb1345f3a375\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375", g.ExecOutput(t, p.root, goBinPath, "list"))
						},
						existingBinaries: []string{"faillint-v1.4.0", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375"},
					},
					{
						name: "Update faillint by name",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "faillint@v1.5.0"))
						},
						existingBinaries: []string{"faillint-v1.4.0", "faillint-v1.5.0", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375"},
					},
					{
						name: "Downgrade faillint by name",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "faillint@v1.3.0"))
						},
						existingBinaries: []string{"faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375"},
					},
					{
						name: "Get another goimports from commit, name it goimports2",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "-n=goimports2", "golang.org/x/tools/cmd/goimports@7d3b6ebf133df879df3e448a8625b7029daa8954"))
							testutil.Equals(t, "Name\t\t\tBinary Name\t\t\t\t\t\t\t\tPackage @ Version\t\t\t\t\t\t\t\t\t\t\t\n----\t\t\t-----------\t\t\t\t\t\t\t\t-----------------\t\t\t\t\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.3.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.3.0\t\t\t\t\t\t\t\t\ngoimports\t\tgoimports-v0.0.0-20200522201501-cb1345f3a375\t\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\t\ngoimports2\tgoimports2-v0.0.0-20200515010526-7d3b6ebf133d\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200515010526-7d3b6ebf133d", g.ExecOutput(t, p.root, goBinPath, "list"))
						},
						existingBinaries: []string{"faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d"},
					},
					{
						name: "Upgrade goimports2 from commit",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "goimports2@7521f6f4253398df2cb300c64dd7fba383ccdfa6"))
						},
						existingBinaries: []string{"faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Get go bindata (non Go Module project).",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "github.com/go-bindata/go-bindata/go-bindata@v3.1.1"))
							testutil.Equals(t, "Name\t\t\tBinary Name\t\t\t\t\t\t\t\tPackage @ Version\t\t\t\t\t\t\t\t\t\t\t\n----\t\t\t-----------\t\t\t\t\t\t\t\t-----------------\t\t\t\t\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.3.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.3.0\t\t\t\t\t\t\t\t\ngo-bindata\tgo-bindata-v3.1.1+incompatible\t\t\t\tgithub.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible\t\t\ngoimports\t\tgoimports-v0.0.0-20200522201501-cb1345f3a375\t\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\t\ngoimports2\tgoimports2-v0.0.0-20200519175826-7521f6f42533\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200519175826-7521f6f42533", g.ExecOutput(t, p.root, goBinPath, "list"))

						},
						existingBinaries: []string{"faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Installing package with `cmd` name fails - different name is suggested.",
						do: func(t *testing.T) {
							testutil.NotOk(t, g.ExectErr(p.root, goBinPath, "get", "github.com/bwplotka/promeval@v0.3.0"))
						},
						existingBinaries: []string{"faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Get array of 4 versions of faillint under f2 name",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "-n", "f2", "github.com/fatih/faillint@v1.5.0,v1.1.0,v1.2.0,v1.0.0"))
							testutil.Equals(t, "Name\tBinary Name\tPackage @ Version\t\t\t\t\n----\t-----------\t-----------------\t\t\t\t\nf2\tf2-v1.5.0\t\tgithub.com/fatih/faillint@v1.5.0\t\nf2\tf2-v1.1.0\t\tgithub.com/fatih/faillint@v1.1.0\t\nf2\tf2-v1.2.0\t\tgithub.com/fatih/faillint@v1.2.0\t\nf2\tf2-v1.0.0\t\tgithub.com/fatih/faillint@v1.0.0", g.ExecOutput(t, p.root, goBinPath, "list", "f2"))
							testutil.Equals(t, "Name\t\t\tBinary Name\t\t\t\t\t\t\t\tPackage @ Version\t\t\t\t\t\t\t\t\t\t\t\n----\t\t\t-----------\t\t\t\t\t\t\t\t-----------------\t\t\t\t\t\t\t\t\t\t\t\nf2\t\t\tf2-v1.5.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.5.0\t\t\t\t\t\t\t\t\nf2\t\t\tf2-v1.1.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.1.0\t\t\t\t\t\t\t\t\nf2\t\t\tf2-v1.2.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.2.0\t\t\t\t\t\t\t\t\nf2\t\t\tf2-v1.0.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.0.0\t\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.3.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.3.0\t\t\t\t\t\t\t\t\ngo-bindata\tgo-bindata-v3.1.1+incompatible\t\t\t\tgithub.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible\t\t\ngoimports\t\tgoimports-v0.0.0-20200522201501-cb1345f3a375\t\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\t\ngoimports2\tgoimports2-v0.0.0-20200519175826-7521f6f42533\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200519175826-7521f6f42533", g.ExecOutput(t, p.root, goBinPath, "list"))
						},
						existingBinaries: []string{"f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.5.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Install package with go name should fail.",
						do: func(t *testing.T) {
							testutil.NotOk(t, g.ExectErr(p.root, goBinPath, "get", "github.com/something/go"))
						},
						existingBinaries: []string{"f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.5.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "<Special> Persist current state, to use for compatibility testing. ",
						do: func(t *testing.T) {
							if isGoProject {
								return
							}
							// Generate current Version test case for further tests. This should be committed as well if changed.
							testutil.Ok(t, os.RemoveAll(currTestCaseDir))
							testutil.Ok(t, os.MkdirAll(filepath.Join(currTestCaseDir, ".bingo"), os.ModePerm))
							_, err := execCmd("", nil, "cp", "-r", filepath.Join(p.root, ".bingo"), currTestCaseDir)
							testutil.Ok(t, err)
						},
						existingBinaries: []string{"f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.5.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Get array of 2 versions of normal faillint, despite being non array before, should work",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "faillint@v1.1.0,v1.0.0"))
						},
						existingBinaries: []string{"f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.5.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Updating f2 to different version should work",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "f2@v1.3.0,v1.4.0"))
						},
						existingBinaries: []string{"f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Updating f2 to same multiple versions should fail",
						do: func(t *testing.T) {
							testutil.NotOk(t, g.ExectErr(p.root, goBinPath, "get", "f2@v1.1.0,v1.4.0,v1.1.0"))
						},
						existingBinaries: []string{"f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Creating not existing foo to f3 should fail",
						do: func(t *testing.T) {
							testutil.NotOk(t, g.ExectErr(p.root, goBinPath, "get", "-n", "f3", "x"))
						},
						existingBinaries: []string{"f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Renaming not existing foo to f3 should fail",
						do: func(t *testing.T) {
							testutil.NotOk(t, g.ExectErr(p.root, goBinPath, "get", "-r", "f3", "x"))
						},
						existingBinaries: []string{"f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Cloning f2 to f2-clone should work",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "-n", "f2-clone", "f2"))
							testutil.Equals(t, "Name\t\t\tBinary Name\t\t\t\t\t\t\t\tPackage @ Version\t\t\t\t\t\t\t\t\t\t\t\n----\t\t\t-----------\t\t\t\t\t\t\t\t-----------------\t\t\t\t\t\t\t\t\t\t\t\nf2-clone\t\tf2-clone-v1.3.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.3.0\t\t\t\t\t\t\t\t\nf2-clone\t\tf2-clone-v1.4.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.4.0\t\t\t\t\t\t\t\t\nf2\t\t\tf2-v1.3.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.3.0\t\t\t\t\t\t\t\t\nf2\t\t\tf2-v1.4.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.4.0\t\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.1.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.1.0\t\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.0.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.0.0\t\t\t\t\t\t\t\t\ngo-bindata\tgo-bindata-v3.1.1+incompatible\t\t\t\tgithub.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible\t\t\ngoimports\t\tgoimports-v0.0.0-20200522201501-cb1345f3a375\t\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\t\ngoimports2\tgoimports2-v0.0.0-20200519175826-7521f6f42533\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200519175826-7521f6f42533", g.ExecOutput(t, p.root, goBinPath, "list"))
						},
						existingBinaries: []string{"f2-clone-v1.3.0", "f2-clone-v1.4.0", "f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Deleting f2-clone",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "f2-clone@none"))
							testutil.Equals(t, "Name\t\t\tBinary Name\t\t\t\t\t\t\t\tPackage @ Version\t\t\t\t\t\t\t\t\t\t\t\n----\t\t\t-----------\t\t\t\t\t\t\t\t-----------------\t\t\t\t\t\t\t\t\t\t\t\nf2\t\t\tf2-v1.3.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.3.0\t\t\t\t\t\t\t\t\nf2\t\t\tf2-v1.4.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.4.0\t\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.1.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.1.0\t\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.0.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.0.0\t\t\t\t\t\t\t\t\ngo-bindata\tgo-bindata-v3.1.1+incompatible\t\t\t\tgithub.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible\t\t\ngoimports\t\tgoimports-v0.0.0-20200522201501-cb1345f3a375\t\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\t\ngoimports2\tgoimports2-v0.0.0-20200519175826-7521f6f42533\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200519175826-7521f6f42533", g.ExecOutput(t, p.root, goBinPath, "list"))
						},
						existingBinaries: []string{"f2-clone-v1.3.0", "f2-clone-v1.4.0", "f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Renaming f2 to f3 should work",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "-r", "f3", "f2"))
							testutil.Equals(t, "Name\t\t\tBinary Name\t\t\t\t\t\t\t\tPackage @ Version\t\t\t\t\t\t\t\t\t\t\t\n----\t\t\t-----------\t\t\t\t\t\t\t\t-----------------\t\t\t\t\t\t\t\t\t\t\t\nf3\t\t\tf3-v1.3.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.3.0\t\t\t\t\t\t\t\t\nf3\t\t\tf3-v1.4.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.4.0\t\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.1.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.1.0\t\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.0.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.0.0\t\t\t\t\t\t\t\t\ngo-bindata\tgo-bindata-v3.1.1+incompatible\t\t\t\tgithub.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible\t\t\ngoimports\t\tgoimports-v0.0.0-20200522201501-cb1345f3a375\t\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\t\ngoimports2\tgoimports2-v0.0.0-20200519175826-7521f6f42533\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200519175826-7521f6f42533", g.ExecOutput(t, p.root, goBinPath, "list"))
						},
						existingBinaries: []string{"f2-clone-v1.3.0", "f2-clone-v1.4.0", "f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "f3-v1.3.0", "f3-v1.4.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Renaming f3 to f4 with certain version should work",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "-r", "f4", "f3@v1.1.0,v1.0.0"))
							testutil.Equals(t, "Name\t\t\tBinary Name\t\t\t\t\t\t\t\tPackage @ Version\t\t\t\t\t\t\t\t\t\t\t\n----\t\t\t-----------\t\t\t\t\t\t\t\t-----------------\t\t\t\t\t\t\t\t\t\t\t\nf4\t\t\tf4-v1.1.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.1.0\t\t\t\t\t\t\t\t\nf4\t\t\tf4-v1.0.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.0.0\t\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.1.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.1.0\t\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.0.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.0.0\t\t\t\t\t\t\t\t\ngo-bindata\tgo-bindata-v3.1.1+incompatible\t\t\t\tgithub.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible\t\t\ngoimports\t\tgoimports-v0.0.0-20200522201501-cb1345f3a375\t\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\t\ngoimports2\tgoimports2-v0.0.0-20200519175826-7521f6f42533\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200519175826-7521f6f42533", g.ExecOutput(t, p.root, goBinPath, "list"))
						},
						existingBinaries: []string{"f2-clone-v1.3.0", "f2-clone-v1.4.0", "f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "f3-v1.3.0", "f3-v1.4.0", "f4-v1.0.0", "f4-v1.1.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Updating f4 to multiple versions with none should fail",
						do: func(t *testing.T) {
							testutil.NotOk(t, g.ExectErr(p.root, goBinPath, "get", "f2@v1.4.0,v1.1.0,none"))
						},
						existingBinaries: []string{"f2-clone-v1.3.0", "f2-clone-v1.4.0", "f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "f3-v1.3.0", "f3-v1.4.0", "f4-v1.0.0", "f4-v1.1.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Updating f4 back to non array version should work",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "f4@v1.1.0"))
						},
						existingBinaries: []string{"f2-clone-v1.3.0", "f2-clone-v1.4.0", "f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "f3-v1.3.0", "f3-v1.4.0", "f4-v1.0.0", "f4-v1.1.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Remove goimports2 by name",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "goimports2@none"))
							testutil.Equals(t, "Name\t\t\tBinary Name\t\t\t\t\t\t\tPackage @ Version\t\t\t\t\t\t\t\t\t\t\n----\t\t\t-----------\t\t\t\t\t\t\t-----------------\t\t\t\t\t\t\t\t\t\t\nf4\t\t\tf4-v1.1.0\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.1.0\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.1.0\t\t\t\t\t\tgithub.com/fatih/faillint@v1.1.0\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.0.0\t\t\t\t\t\tgithub.com/fatih/faillint@v1.0.0\t\t\t\t\t\t\t\ngo-bindata\tgo-bindata-v3.1.1+incompatible\t\t\tgithub.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible\t\ngoimports\t\tgoimports-v0.0.0-20200522201501-cb1345f3a375\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375", g.ExecOutput(t, p.root, goBinPath, "list"))
						},
						existingBinaries: []string{"f2-clone-v1.3.0", "f2-clone-v1.4.0", "f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "f3-v1.3.0", "f3-v1.4.0", "f4-v1.0.0", "f4-v1.1.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Remove goimports by path",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "golang.org/x/tools/cmd/goimports@none"))
						},
						existingBinaries: []string{"f2-clone-v1.3.0", "f2-clone-v1.4.0", "f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "f3-v1.3.0", "f3-v1.4.0", "f4-v1.0.0", "f4-v1.1.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Remove faillint by name",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "faillint@none"))
						},
						existingBinaries: []string{"f2-clone-v1.3.0", "f2-clone-v1.4.0", "f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "f3-v1.3.0", "f3-v1.4.0", "f4-v1.0.0", "f4-v1.1.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Remove f4 by name",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "f4@none"))
						},
						existingBinaries: []string{"f2-clone-v1.3.0", "f2-clone-v1.4.0", "f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "f3-v1.3.0", "f3-v1.4.0", "f4-v1.0.0", "f4-v1.1.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Remove go-bindata by name",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "go-bindata@none"))
							testutil.Equals(t, "Name\tBinary Name\tPackage @ Version\t\n----\t-----------\t-----------------", g.ExecOutput(t, p.root, goBinPath, "list"))
						},
						existingBinaries: []string{"f2-clone-v1.3.0", "f2-clone-v1.4.0", "f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "f3-v1.3.0", "f3-v1.4.0", "f4-v1.0.0", "f4-v1.1.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
				} {
					if ok := t.Run(tcase.name, func(t *testing.T) {
						defer p.assertNotChanged(t, defaultModDir)

						tcase.do(t)
						testutil.Equals(t, tcase.existingBinaries, g.existingBinaries(t))
					}); !ok {
						return
					}
				}
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
						t.Run("Via bingo get all", func(t *testing.T) {
							g := newTmpGoEnv(t)
							defer g.Close(t)

							// We manually build bingo binary to make sure GOCACHE will not hit us.
							goBinPath := filepath.Join(g.tmpDir, bingoBin)
							buildInitialGobin(t, goBinPath)

							// Copy testproject at the beginning to temp dir.
							p := newTestProject(t, dir, filepath.Join(g.tmpDir, "testproject1"), isGoProject)
							p.assertNotChanged(t, defaultModDir)

							testutil.Equals(t, []string{}, g.existingBinaries(t))

							testutil.Equals(t, "Name\t\t\tBinary Name\t\t\t\t\t\t\t\tPackage @ Version\t\t\t\t\t\t\t\t\t\t\t\n----\t\t\t-----------\t\t\t\t\t\t\t\t-----------------\t\t\t\t\t\t\t\t\t\t\t\nf2\t\t\tf2-v1.5.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.5.0\t\t\t\t\t\t\t\t\nf2\t\t\tf2-v1.1.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.1.0\t\t\t\t\t\t\t\t\nf2\t\t\tf2-v1.2.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.2.0\t\t\t\t\t\t\t\t\nf2\t\t\tf2-v1.0.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.0.0\t\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.3.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.3.0\t\t\t\t\t\t\t\t\ngo-bindata\tgo-bindata-v3.1.1+incompatible\t\t\t\tgithub.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible\t\t\ngoimports\t\tgoimports-v0.0.0-20200522201501-cb1345f3a375\t\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\t\ngoimports2\tgoimports2-v0.0.0-20200519175826-7521f6f42533\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200519175826-7521f6f42533", g.ExecOutput(t, p.root, goBinPath, "list"))
							defer p.assertNotChanged(t, defaultModDir)

							// Get all binaries by doing 'bingo get'.
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get"))
							testutil.Equals(t, []string{"f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.5.0", "faillint-v1.3.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200519175826-7521f6f42533"}, g.existingBinaries(t))
							testutil.Equals(t, "Name\t\t\tBinary Name\t\t\t\t\t\t\t\tPackage @ Version\t\t\t\t\t\t\t\t\t\t\t\n----\t\t\t-----------\t\t\t\t\t\t\t\t-----------------\t\t\t\t\t\t\t\t\t\t\t\nf2\t\t\tf2-v1.5.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.5.0\t\t\t\t\t\t\t\t\nf2\t\t\tf2-v1.1.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.1.0\t\t\t\t\t\t\t\t\nf2\t\t\tf2-v1.2.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.2.0\t\t\t\t\t\t\t\t\nf2\t\t\tf2-v1.0.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.0.0\t\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.3.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.3.0\t\t\t\t\t\t\t\t\ngo-bindata\tgo-bindata-v3.1.1+incompatible\t\t\t\tgithub.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible\t\t\ngoimports\t\tgoimports-v0.0.0-20200522201501-cb1345f3a375\t\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\t\ngoimports2\tgoimports2-v0.0.0-20200519175826-7521f6f42533\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200519175826-7521f6f42533", g.ExecOutput(t, p.root, goBinPath, "list"))

						})
						t.Run("Via bingo get one by one", func(t *testing.T) {
							g := newTmpGoEnv(t)
							defer g.Close(t)

							// We manually build bingo binary to make sure GOCACHE will not hit us.
							goBinPath := filepath.Join(g.tmpDir, bingoBin)
							buildInitialGobin(t, goBinPath)

							// Copy testproject at the beginning to temp dir.
							p := newTestProject(t, dir, filepath.Join(g.tmpDir, "testproject1"), isGoProject)
							p.assertNotChanged(t, defaultModDir)

							testutil.Equals(t, []string{}, g.existingBinaries(t))
							defer p.assertNotChanged(t, defaultModDir)

							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "faillint"))
							testutil.Equals(t, []string{"faillint-v1.3.0"}, g.existingBinaries(t))
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "goimports"))
							testutil.Equals(t, []string{"faillint-v1.3.0", "goimports-v0.0.0-20200522201501-cb1345f3a375"}, g.existingBinaries(t))
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "goimports2"))
							testutil.Equals(t, []string{"faillint-v1.3.0", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200519175826-7521f6f42533"}, g.existingBinaries(t))
							// Get array version with one go.
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "f2"))
							testutil.Equals(t, []string{"f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.5.0", "faillint-v1.3.0", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200519175826-7521f6f42533"}, g.existingBinaries(t))
							testutil.Equals(t, "Name\t\t\tBinary Name\t\t\t\t\t\t\t\tPackage @ Version\t\t\t\t\t\t\t\t\t\t\t\n----\t\t\t-----------\t\t\t\t\t\t\t\t-----------------\t\t\t\t\t\t\t\t\t\t\t\nf2\t\t\tf2-v1.5.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.5.0\t\t\t\t\t\t\t\t\nf2\t\t\tf2-v1.1.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.1.0\t\t\t\t\t\t\t\t\nf2\t\t\tf2-v1.2.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.2.0\t\t\t\t\t\t\t\t\nf2\t\t\tf2-v1.0.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.0.0\t\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.3.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.3.0\t\t\t\t\t\t\t\t\ngo-bindata\tgo-bindata-v3.1.1+incompatible\t\t\t\tgithub.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible\t\t\ngoimports\t\tgoimports-v0.0.0-20200522201501-cb1345f3a375\t\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\t\ngoimports2\tgoimports2-v0.0.0-20200519175826-7521f6f42533\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200519175826-7521f6f42533", g.ExecOutput(t, p.root, goBinPath, "list"))
						})
						t.Run("Via go", func(t *testing.T) {
							g := newTmpGoEnv(t)
							defer g.Close(t)

							// Copy testproject at the beginning to temp dir.
							// NOTE: No bingo binary is required here.
							p := newTestProject(t, dir, filepath.Join(g.tmpDir, "testproject2"), isGoProject)
							p.assertNotChanged(t, defaultModDir)

							testutil.Equals(t, []string{}, g.existingBinaries(t))
							defer p.assertNotChanged(t, defaultModDir)

							// Get all binaries by doing native go build.
							if isGoProject {
								// This should work without cd even.
								_, err := execCmd(p.root, nil, "go", "build", "-modfile="+filepath.Join(defaultModDir, "goimports.mod"), "-o="+filepath.Join(g.gobin, "goimports-v0.0.0-20200522201501-cb1345f3a375"), "golang.org/x/tools/cmd/goimports")
								testutil.Ok(t, err)
								_, err = execCmd(p.root, nil, "go", "build", "-modfile="+filepath.Join(defaultModDir, "faillint.mod"), "-o="+filepath.Join(g.gobin, "faillint-v1.3.0"), "github.com/fatih/faillint")
								testutil.Ok(t, err)
								_, err = execCmd(p.root, nil, "go", "build", "-modfile="+filepath.Join(defaultModDir, "goimports2.mod"), "-o="+filepath.Join(g.gobin, "goimports2-v0.0.0-20200519175826-7521f6f42533"), "golang.org/x/tools/cmd/goimports")
								testutil.Ok(t, err)
							} else {
								// For no go projects we have this "bug" that requires go.mod to be present.
								_, err := execCmd(filepath.Join(p.root, defaultModDir), nil, "go", "build", "-modfile=goimports.mod", "-o="+filepath.Join(g.gobin, "goimports-v0.0.0-20200522201501-cb1345f3a375"), "golang.org/x/tools/cmd/goimports")
								testutil.Ok(t, err)
								_, err = execCmd(filepath.Join(p.root, defaultModDir), nil, "go", "build", "-modfile=faillint.mod", "-o="+filepath.Join(g.gobin, "faillint-v1.3.0"), "github.com/fatih/faillint")
								testutil.Ok(t, err)
								_, err = execCmd(filepath.Join(p.root, defaultModDir), nil, "go", "build", "-modfile=goimports2.mod", "-o="+filepath.Join(g.gobin, "goimports2-v0.0.0-20200519175826-7521f6f42533"), "golang.org/x/tools/cmd/goimports")
								testutil.Ok(t, err)
							}
							testutil.Equals(t, []string{"faillint-v1.3.0", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200519175826-7521f6f42533"}, g.existingBinaries(t))
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

							testutil.Equals(t, []string{}, g.existingBinaries(t))
							testutil.Equals(t, "(re)installing "+g.gobin+"/faillint-v1.3.0\ngo: downloading github.com/fatih/faillint v1.3.0\ngo: downloading golang.org/x/tools v0.0.0-20200207224406-61798d64f025\nchecking faillint\n", g.ExecOutput(t, p.root, makePath, "faillint-exists"))
							testutil.Equals(t, "(re)installing "+g.gobin+"/goimports-v0.0.0-20200522201501-cb1345f3a375\ngo: downloading golang.org/x/tools v0.0.0-20200522201501-cb1345f3a375\ngo: downloading golang.org/x/mod v0.2.0\ngo: downloading golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543\nchecking goimports\n", g.ExecOutput(t, p.root, makePath, "goimports-exists"))
							testutil.Equals(t, "(re)installing "+g.gobin+"/goimports2-v0.0.0-20200519175826-7521f6f42533\ngo: downloading golang.org/x/tools v0.0.0-20200519175826-7521f6f42533\nchecking goimports2\n", g.ExecOutput(t, p.root, makePath, "goimports2-exists"))

							testutil.Equals(t, "checking faillint\n", g.ExecOutput(t, p.root, makePath, "faillint-exists"))
							testutil.Equals(t, "checking goimports\n", g.ExecOutput(t, p.root, makePath, "goimports-exists"))
							testutil.Equals(t, "checking goimports2\n", g.ExecOutput(t, p.root, makePath, "goimports2-exists"))

							testutil.Equals(t, []string{"faillint-v1.3.0", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200519175826-7521f6f42533"}, g.existingBinaries(t))
							t.Run("Delete binary file, expect reinstall", func(t *testing.T) {
								_, err := execCmd(g.gobin, nil, "rm", "faillint-v1.3.0")
								testutil.Ok(t, err)
								testutil.Equals(t, []string{"goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200519175826-7521f6f42533"}, g.existingBinaries(t))

								testutil.Equals(t, "(re)installing "+g.gobin+"/faillint-v1.3.0\nchecking faillint\n", g.ExecOutput(t, p.root, makePath, "faillint-exists"))
								testutil.Equals(t, "checking faillint\n", g.ExecOutput(t, p.root, makePath, "faillint-exists"))
								testutil.Equals(t, "checking goimports\n", g.ExecOutput(t, p.root, makePath, "goimports-exists"))
								testutil.Equals(t, "checking goimports2\n", g.ExecOutput(t, p.root, makePath, "goimports2-exists"))
								testutil.Equals(t, []string{"faillint-v1.3.0", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200519175826-7521f6f42533"}, g.existingBinaries(t))
							})
							t.Run("Delete makefile", func(t *testing.T) {
								fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "f2@none"))
								fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "faillint@none"))
								fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "goimports@none"))
								fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "goimports2@none"))
								fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "go-bindata@none"))

								testutil.Equals(t, "Name\tBinary Name\tPackage @ Version\t\n----\t-----------\t-----------------", g.ExecOutput(t, p.root, goBinPath, "list"))

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
		// Make sure we don't require clang to build etc.
		fmt.Sprintf("CGO_ENABLED=0"),
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

func (g *goEnv) existingBinaries(t *testing.T) []string {
	var filenames []string
	files, err := ioutil.ReadDir(g.gobin)
	if os.IsNotExist(err) {
		return []string{}
	}
	testutil.Ok(t, err)

	for _, f := range files {
		if f.IsDir() {
			t.Fatal("Did not expect directory in gobin", g.gobin)
		}
		filenames = append(filenames, f.Name())
	}
	return filenames
}

func (g *goEnv) Close(t testing.TB) {
	_, err := execCmd("", nil, "chmod", "-R", "777", g.tmpDir)
	testutil.Ok(t, err)
	testutil.Ok(t, os.RemoveAll(g.tmpDir))
}
