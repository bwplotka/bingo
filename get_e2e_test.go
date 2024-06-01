// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package main_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/bwplotka/bingo/pkg/bingo"
	"github.com/bwplotka/bingo/pkg/runner"
	"github.com/bwplotka/bingo/pkg/version"
	"github.com/efficientgo/core/testutil"
)

const (
	defaultModDir  = ".bingo"
	defaultGoProxy = "https://proxy.golang.org"
)

func TestGetList(t *testing.T) {
	t.Parallel()

	g := newIsolatedGoEnv(t, defaultGoProxy)
	defer g.Close(t)

	for _, isGoProject := range []bool{false, true} {
		if ok := t.Run(fmt.Sprintf("isGoProject=%v", isGoProject), func(t *testing.T) {
			g.Clear(t)

			testutil.Ok(t, os.MkdirAll(filepath.Join(g.tmpDir, "newproject"), os.ModePerm))
			p := newTestProject(t, filepath.Join(g.tmpDir, "newproject"), filepath.Join(g.tmpDir, "testproject"), isGoProject)
			p.assertNotChanged(t)

			// We manually build bingo binary to make sure GOCACHE will not hit us.
			bingoPath := filepath.Join(g.tmpDir, bingoBin)
			buildInitialGobin(t, bingoPath)

			// TODO(bwplotka): Add buildable back from array version to normal one.
			var prevBinaries []string
			for _, tcase := range []struct {
				name string
				do   func(t *testing.T)

				expectBinaries             []string
				expectSameBinariesAsBefore bool
				expectRows                 []row
			}{
				{
					name: "get one module by tag on empty project",
					do: func(t *testing.T) {
						fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "github.com/bwplotka/bingo-testmodule/buildable@v1.0.0"))
						testutil.Equals(t, g.ExecOutput(t, p.root, bingoPath, "list", "buildable"), g.ExecOutput(t, p.root, bingoPath, "list"))
					},
					expectRows:     []row{{name: "buildable", binName: "buildable-v1.0.0", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable@v1.0.0"}},
					expectBinaries: []string{"buildable-v1.0.0"},
				},
				{
					name: "get the same module by tag does not change anything",
					do: func(t *testing.T) {
						fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "github.com/bwplotka/bingo-testmodule/buildable@v1.0.0"))
						testutil.Equals(t, g.ExecOutput(t, p.root, bingoPath, "list", "buildable"), g.ExecOutput(t, p.root, bingoPath, "list"))
					},
					expectRows:                 []row{{name: "buildable", binName: "buildable-v1.0.0", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable@v1.0.0"}},
					expectSameBinariesAsBefore: true,
				},
				{
					name: "get the same module by commit does not change anything",
					do: func(t *testing.T) {
						fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "github.com/bwplotka/bingo-testmodule/buildable@4040b74bfd07be764f2ad16ddbf7fba07b247f1c"))
						testutil.Equals(t, g.ExecOutput(t, p.root, bingoPath, "list", "buildable"), g.ExecOutput(t, p.root, bingoPath, "list"))
					},
					expectRows:                 []row{{name: "buildable", binName: "buildable-v1.0.0", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable@v1.0.0"}},
					expectSameBinariesAsBefore: true,
				},
				{
					name: "get the different module with different tool name by commit",
					do: func(t *testing.T) {
						fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "github.com/bwplotka/bingo-testmodule/buildable2@9d83f47b84c5d9262ecaf649bfa01f0f1cb6ebd2"))
					},
					expectRows: []row{
						{name: "buildable", binName: "buildable-v1.0.0", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable@v1.0.0"},
						{name: "buildable2", binName: "buildable2-v0.0.0-20221007091238-9d83f47b84c5", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable2@v0.0.0-20221007091238-9d83f47b84c5"},
					},
					expectBinaries: []string{"buildable-v1.0.0", "buildable2-v0.0.0-20221007091238-9d83f47b84c5"},
				},
				{
					name: "latest tag should upgrade the tool.",
					do: func(t *testing.T) {
						fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "buildable@latest"))
					},
					expectRows: []row{
						{name: "buildable", binName: "buildable-v1.1.0", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable@v1.1.0"},
						{name: "buildable2", binName: "buildable2-v0.0.0-20221007091238-9d83f47b84c5", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable2@v0.0.0-20221007091238-9d83f47b84c5"},
					},
					expectBinaries: []string{"buildable-v1.0.0", "buildable-v1.1.0", "buildable2-v0.0.0-20221007091238-9d83f47b84c5"},
				},
				{
					name: "downgrade the tool",
					do: func(t *testing.T) {
						fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "github.com/bwplotka/bingo-testmodule/buildable@v1.0.0"))
					},
					expectRows: []row{
						{name: "buildable", binName: "buildable-v1.0.0", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable@v1.0.0"},
						{name: "buildable2", binName: "buildable2-v0.0.0-20221007091238-9d83f47b84c5", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable2@v0.0.0-20221007091238-9d83f47b84c5"},
					},
					expectSameBinariesAsBefore: true,
				},
				{
					name: "get same tool under different name",
					do: func(t *testing.T) {
						fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "-n", "buildable-v2", "github.com/bwplotka/bingo-testmodule/v2/buildable@v2.0.0"))
					},
					expectRows: []row{
						{name: "buildable", binName: "buildable-v1.0.0", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable@v1.0.0"},
						{name: "buildable-v2", binName: "buildable-v2-v2.0.0", pkgVersion: "github.com/bwplotka/bingo-testmodule/v2/buildable@v2.0.0"},
						{name: "buildable2", binName: "buildable2-v0.0.0-20221007091238-9d83f47b84c5", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable2@v0.0.0-20221007091238-9d83f47b84c5"},
					},
					expectBinaries: []string{"buildable-v1.0.0", "buildable-v1.1.0", "buildable-v2-v2.0.0", "buildable2-v0.0.0-20221007091238-9d83f47b84c5"},
				},
				{
					name: "rename the tool",
					do: func(t *testing.T) {
						fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "-r", "my-buildable-v2", "buildable-v2"))
					},
					expectRows: []row{
						{name: "buildable", binName: "buildable-v1.0.0", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable@v1.0.0"},
						{name: "buildable2", binName: "buildable2-v0.0.0-20221007091238-9d83f47b84c5", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable2@v0.0.0-20221007091238-9d83f47b84c5"},
						{name: "my-buildable-v2", binName: "my-buildable-v2-v2.0.0", pkgVersion: "github.com/bwplotka/bingo-testmodule/v2/buildable@v2.0.0"},
					},
					expectBinaries: []string{"buildable-v1.0.0", "buildable-v1.1.0", "buildable-v2-v2.0.0", "buildable2-v0.0.0-20221007091238-9d83f47b84c5", "my-buildable-v2-v2.0.0"},
				},
				{
					name: "build as latest",
					do: func(t *testing.T) {
						fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "-l", "buildable"))
					},
					expectRows: []row{
						{name: "buildable", binName: "buildable-v1.0.0", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable@v1.0.0"},
						{name: "buildable2", binName: "buildable2-v0.0.0-20221007091238-9d83f47b84c5", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable2@v0.0.0-20221007091238-9d83f47b84c5"},
						{name: "my-buildable-v2", binName: "my-buildable-v2-v2.0.0", pkgVersion: "github.com/bwplotka/bingo-testmodule/v2/buildable@v2.0.0"},
					},
					expectBinaries: []string{"buildable", "buildable-v1.0.0", "buildable-v1.1.0", "buildable-v2-v2.0.0", "buildable2-v0.0.0-20221007091238-9d83f47b84c5", "my-buildable-v2-v2.0.0"},
				},
				{
					name: "remove tool",
					do: func(t *testing.T) {
						fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "buildable@none"))
					},
					expectRows: []row{
						{name: "buildable2", binName: "buildable2-v0.0.0-20221007091238-9d83f47b84c5", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable2@v0.0.0-20221007091238-9d83f47b84c5"},
						{name: "my-buildable-v2", binName: "my-buildable-v2-v2.0.0", pkgVersion: "github.com/bwplotka/bingo-testmodule/v2/buildable@v2.0.0"},
					},
					expectSameBinariesAsBefore: true,
				},
				{
					name: "install array version",
					do: func(t *testing.T) {
						fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "github.com/bwplotka/bingo-testmodule/buildable@39a7f0ae0b1e1e67a75033fc671ccc2c5b3bbddf,v1.0.0,v1.1.0"))
					},
					expectRows: []row{
						{name: "buildable", binName: "buildable-v0.0.0-20221007091146-39a7f0ae0b1e", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable@v0.0.0-20221007091146-39a7f0ae0b1e"},
						{name: "buildable", binName: "buildable-v1.0.0", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable@v1.0.0"},
						{name: "buildable", binName: "buildable-v1.1.0", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable@v1.1.0"},
						{name: "buildable2", binName: "buildable2-v0.0.0-20221007091238-9d83f47b84c5", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable2@v0.0.0-20221007091238-9d83f47b84c5"},
						{name: "my-buildable-v2", binName: "my-buildable-v2-v2.0.0", pkgVersion: "github.com/bwplotka/bingo-testmodule/v2/buildable@v2.0.0"},
					},
					expectBinaries: []string{"buildable", "buildable-v0.0.0-20221007091146-39a7f0ae0b1e", "buildable-v1.0.0", "buildable-v1.1.0", "buildable-v2-v2.0.0", "buildable2-v0.0.0-20221007091238-9d83f47b84c5", "my-buildable-v2-v2.0.0"},
				},
				{
					name: "get from array version to single one",
					do: func(t *testing.T) {
						fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "buildable@v1.1.0"))
					},
					expectRows: []row{
						{name: "buildable", binName: "buildable-v1.1.0", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable@v1.1.0"},
						{name: "buildable2", binName: "buildable2-v0.0.0-20221007091238-9d83f47b84c5", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable2@v0.0.0-20221007091238-9d83f47b84c5"},
						{name: "my-buildable-v2", binName: "my-buildable-v2-v2.0.0", pkgVersion: "github.com/bwplotka/bingo-testmodule/v2/buildable@v2.0.0"},
					},
					expectSameBinariesAsBefore: true,
				},
				{
					name: "get from single version back to array",
					do: func(t *testing.T) {
						fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "github.com/bwplotka/bingo-testmodule/buildable@39a7f0ae0b1e1e67a75033fc671ccc2c5b3bbddf,v1.0.0,v1.1.0"))
					},
					expectRows: []row{
						{name: "buildable", binName: "buildable-v0.0.0-20221007091146-39a7f0ae0b1e", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable@v0.0.0-20221007091146-39a7f0ae0b1e"},
						{name: "buildable", binName: "buildable-v1.0.0", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable@v1.0.0"},
						{name: "buildable", binName: "buildable-v1.1.0", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable@v1.1.0"},
						{name: "buildable2", binName: "buildable2-v0.0.0-20221007091238-9d83f47b84c5", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable2@v0.0.0-20221007091238-9d83f47b84c5"},
						{name: "my-buildable-v2", binName: "my-buildable-v2-v2.0.0", pkgVersion: "github.com/bwplotka/bingo-testmodule/v2/buildable@v2.0.0"},
					},
					expectSameBinariesAsBefore: true,
				},
				{
					name: "remove all",
					do: func(t *testing.T) {
						fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "buildable@none"))
						fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "buildable2@none"))
						fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "my-buildable-v2@none"))

						// Check additionally if Makefile was removed.
						_, err := os.Stat(filepath.Join(p.root, ".bingo", "Variables.mk"))
						testutil.NotOk(t, err)
					},
					expectSameBinariesAsBefore: true,
				},
			} {
				if ok := t.Run(tcase.name, func(t *testing.T) {
					defer p.assertNotChanged(t, defaultModDir)

					tcase.do(t)

					expectBingoListRows(t, tcase.expectRows, g.ExecOutput(t, p.root, bingoPath, "list"))

					binaries := g.existingBinaries(t)
					if tcase.expectSameBinariesAsBefore {
						testutil.Equals(t, prevBinaries, g.existingBinaries(t))
					} else {
						testutil.Equals(t, tcase.expectBinaries, g.existingBinaries(t))
					}
					prevBinaries = binaries
				}); !ok {
					return
				}
			}

		}); !ok {
			return
		}
	}
}

func TestGet_Errors(t *testing.T) {
	// Existing bug - > on names can be case-sensitive but makefile varialbes are not!

	// 	name: "error cases",
	//						do: func(t *testing.T) {
	//							// Installing different tool with name clash should fail
	//							testutil.NotOk(t, g.ExpectErr(p.root, bingoPath, "get", "github.com/bwplotka/totally-not-bingo/testdata/module/buildable2@v0.0.0-20210109093942-2e6391144e85"))
	//							// Installing package with go name should fail. (this is due to clash with go.mod).
	//							testutil.NotOk(t, g.ExpectErr(p.root, bingoPath, "get", "github.com/something/go"))
	//							// Naive installing package that would result with `cmd` name fails - different name is suggested.
	//							testutil.NotOk(t, g.ExpectErr(p.root, bingoPath, "get", "github.com/bwplotka/promeval@v0.3.0"))
	//							// Updating f4 to multiple versions with none should fail.
	//							testutil.NotOk(t, g.ExpectErr(p.root, bingoPath, "get", "f2@v1.4.0,v1.1.0,none"))
	//							// Installing by different path that would result in same name
	//							testutil.NotOk(t, g.ExpectErr(p.root, bingoPath, "get", "github.com/bwplotka/bingo/some/module/buildable"))
	//							// Removing by path.
	//							testutil.NotOk(t, g.ExpectErr(p.root, bingoPath, "get", "github.com/bwplotka/bingo/testdata/module/buildable@none"))
	//							testutil.NotOk(t, g.ExpectErr(p.root, bingoPath, "get", "github.com/bwplotka/bingo/some/module/buildable@none"))
	//							// Removing non existing tool.
	//							testutil.NotOk(t, g.ExpectErr(p.root, bingoPath, "get", "buildable2@none"))
	//						},
	// Get unexisting module
	// Rename unexisting tool
	// Removing unexisting tool
	// Rm by path
	// Use g.ExpectErr(p...
	// 	name: "-r rename error cases",
	//						do: func(t *testing.T) {
	//							testutil.NotOk(t, g.ExpectErr(p.root, bingoPath, "get", "-r=buildable4", "github.com/bwplotka/bingo/testdata/module/buildable2"))
	//							testutil.NotOk(t, g.ExpectErr(p.root, bingoPath, "get", "-r=buildable4", "github.com/bwplotka/bingo/testdata/module/buildable2@v0.0.0-20210109093942-2e6391144e85"))
	//							testutil.NotOk(t, g.ExpectErr(p.root, bingoPath, "get", "-r=buildable4", "github.com/bwplotka/bingo/testdata/module/buildable2@none"))
	//							testutil.NotOk(t, g.ExpectErr(p.root, bingoPath, "get", "-r=buildable4", "buildable2@none"))
	//							testutil.NotOk(t, g.ExpectErr(p.root, bingoPath, "get", "-r=buildable4", "buildable2@v0.0.0-20210109093942-2e6391144e85"))
	//							testutil.NotOk(t, g.ExpectErr(p.root, bingoPath, "get", "-r=faillint", "buildable2")) // Renaming to existing name.
	//							testutil.NotOk(t, g.ExpectErr(p.root, bingoPath, "get", "-r=faiLLint", "buildable2")) // Renaming to existing name (it's not case sensitive).
	//							testutil.NotOk(t, g.ExpectErr(p.root, bingoPath, "get", "-r", "f3", "x"))             // Renaming not existing.
	//							testutil.NotOk(t, g.ExpectErr(p.root, bingoPath, "get", "-r", "f4", "f3@v1.1.0,v1.0.0"))
	//						},
	// 	{
	//						name: "get wr_buildable@ccbd4039b94aac79d926ba5eebfe6a132a728ed8 (dowgrade buildable with different replaces - trickier than you think!)",
	//						do: func(t *testing.T) {
	//							fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "wr_buildable@ccbd4039b94aac79d926ba5eebfe6a132a728ed8"))
	//
	//							// Check if installed tool is what we expect.
	//							testutil.Equals(t, "module_with_replace.buildable 2.8\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "wr_buildable-v0.0.0-20210110214650-ab990d1be30b")))
	//							testutil.Equals(t, "module_with_replace.buildable 2.7\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "wr_buildable-v0.0.0-20210109165512-ccbd4039b94a")))
	//						},
	//						expectRows: []row{
	//							{name: "buildable", binName: "buildable-v0.0.0-20210109094001-375d0606849d", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109094001-375d0606849d"},
	//							{name: "buildable2", binName: "buildable2-v0.0.0-20210109093942-2e6391144e85", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable2@v0.0.0-20210109093942-2e6391144e85"},
	//							{name: "buildable_old", binName: "buildable_old-v0.0.0-20210109093942-2e6391144e85", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109093942-2e6391144e85"},
	//							{name: "faillint", binName: "faillint-v1.3.0", pkgVersion: "github.com/fatih/faillint@v1.3.0"},
	//							{name: "go-bindata", binName: "go-bindata-v3.1.1+incompatible", pkgVersion: "github.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible"},
	//							{name: "wr_buildable", binName: "wr_buildable-v0.0.0-20210109165512-ccbd4039b94a", pkgVersion: "github.com/bwplotka/bingo/testdata/module_with_replace/buildable@v0.0.0-20210109165512-ccbd4039b94a"},
	//						},
	//						expectBinaries: []string{
	//							"buildable-v0.0.0-20210109093942-2e6391144e85", "buildable-v0.0.0-20210109094001-375d0606849d", "buildable2-v0.0.0-20210109093942-2e6391144e85",
	//							"buildable_old-v0.0.0-20210109093942-2e6391144e85", "buildable_old-v0.0.0-20210109094001-375d0606849d",
	//							"faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0",
	//							"go-bindata-v3.1.1+incompatible",
	//							"wr_buildable-v0.0.0-20210109165512-ccbd4039b94a", "wr_buildable-v0.0.0-20210110214650-ab990d1be30b",
	//						},
	//					},
}

func TestCompatibilityCurrentVersionCreate(t *testing.T) {
	currTestCaseDir := fmt.Sprintf("testdata/testproject_with_bingo_%s", strings.ReplaceAll(version.Version, ".", "_"))

	g := newIsolatedGoEnv(t, defaultGoProxy)
	defer g.Close(t)

	// We manually build bingo binary to make sure GOCACHE will not hit us.
	bingoPath := filepath.Join(g.tmpDir, bingoBin)
	buildInitialGobin(t, bingoPath)

	testutil.Ok(t, os.MkdirAll(filepath.Join(g.tmpDir, "newproject"), os.ModePerm))
	p := newTestProject(t, filepath.Join(g.tmpDir, "newproject"), filepath.Join(g.tmpDir, "testproject"), false)
	p.assertNotChanged(t)

	fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "github.com/bwplotka/bingo-testmodule/buildable@39a7f0ae0b1e1e67a75033fc671ccc2c5b3bbddf,v1.0.0,v1.1.0"))
	fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "github.com/bwplotka/bingo-testmodule/buildable2@9d83f47b84c5d9262ecaf649bfa01f0f1cb6ebd2"))
	fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "-n", "buildable-v2", "github.com/bwplotka/bingo-testmodule/v2/buildable@v2.0.0"))
	fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "-n", "buildable-withReplace", "github.com/bwplotka/bingo-testmodule/buildable@fe4d42a37d927cbf6d2c0f30fe1459d493146664"))

	expectBingoListRows(t, bingoExpectedCompatibilityRows, g.ExecOutput(t, p.root, bingoPath, "list"))
	testutil.Equals(t, bingoExpectedCompatibilityBinaries, g.existingBinaries(t))

	// Generate current version test case for further tests. This should be committed as well if changed.
	testutil.Ok(t, os.RemoveAll(currTestCaseDir))
	testutil.Ok(t, os.MkdirAll(filepath.Join(currTestCaseDir, ".bingo"), os.ModePerm))
	_, err := execCmd("", nil, "cp", "-r", filepath.Join(p.root, ".bingo"), currTestCaseDir)
	testutil.Ok(t, err)
}

var (
	bingoExpectedCompatibilityRows = []row{
		{name: "buildable", binName: "buildable-v0.0.0-20221007091146-39a7f0ae0b1e", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable@v0.0.0-20221007091146-39a7f0ae0b1e"},
		{name: "buildable", binName: "buildable-v1.0.0", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable@v1.0.0"},
		{name: "buildable", binName: "buildable-v1.1.0", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable@v1.1.0"},
		{name: "buildable-v2", binName: "buildable-v2-v2.0.0", pkgVersion: "github.com/bwplotka/bingo-testmodule/v2/buildable@v2.0.0"},
		{name: "buildable-withReplace", binName: "buildable-withReplace-v0.0.0-20221007091003-fe4d42a37d92", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable@v0.0.0-20221007091003-fe4d42a37d92"},
		{name: "buildable2", binName: "buildable2-v0.0.0-20221007091238-9d83f47b84c5", pkgVersion: "github.com/bwplotka/bingo-testmodule/buildable2@v0.0.0-20221007091238-9d83f47b84c5"},
	}
	bingoExpectedCompatibilityBinaries = []string{
		"buildable-v0.0.0-20221007091146-39a7f0ae0b1e", "buildable-v1.0.0", "buildable-v1.1.0", "buildable-v2-v2.0.0", "buildable-withReplace-v0.0.0-20221007091003-fe4d42a37d92", "buildable2-v0.0.0-20221007091238-9d83f47b84c5",
	}
)

func TestCompatibility(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip() // bingo for Windows won't be compatible with past versions
	}
	t.Parallel()

	dirs, err := filepath.Glob("testdata/testproject*")
	testutil.Ok(t, err)

	g := newIsolatedGoEnv(t, defaultGoProxy)
	defer g.Close(t)

	var goVersion *semver.Version
	{
		r, err := runner.NewRunner(context.Background(), nil, false, "go")
		testutil.Ok(t, err)
		goVersion = r.GoVersion()
	}

	for _, dir := range dirs {
		t.Run(dir, func(t *testing.T) {
			for _, isGoProject := range []bool{false, true} {
				t.Run(fmt.Sprintf("isGoProject=%v", isGoProject), func(t *testing.T) {
					t.Run("Via bingo get all", func(t *testing.T) {
						g.Clear(t)

						// Copy testproject at the beginning to temp dir.
						p := newTestProject(t, dir, filepath.Join(g.tmpDir, "testproject1"), isGoProject)
						p.assertNotChanged(t, defaultModDir)

						// We manually build bingo binary to make sure GOCACHE will not hit us.
						bingoPath := filepath.Join(g.tmpDir, bingoBin)
						buildInitialGobin(t, bingoPath)

						expectBingoListRows(t, bingoExpectedCompatibilityRows, g.ExecOutput(t, p.root, bingoPath, "list"))
						testutil.Equals(t, []string{}, g.existingBinaries(t))

						defer p.assertNotChanged(t, defaultModDir)

						// Get all binaries by doing 'bingo get'.
						fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get"))
						expectBingoListRows(t, bingoExpectedCompatibilityRows, g.ExecOutput(t, p.root, bingoPath, "list"))
						testutil.Equals(t, bingoExpectedCompatibilityBinaries, g.existingBinaries(t))

						// Expect binaries works:
						// TODO(bwplotka) Check all.
						testutil.Equals(t, "buildable\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "buildable-v1.0.0")))
						testutil.Equals(t, "buildable\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "buildable-withReplace-v0.0.0-20221007091003-fe4d42a37d92")))
					})
					t.Run("Via go", func(t *testing.T) {
						g.Clear(t)

						// Copy testproject at the beginning to temp dir.
						// NOTE: No bingo binary is required here.
						p := newTestProject(t, dir, filepath.Join(g.tmpDir, "testproject2"), isGoProject)
						p.assertNotChanged(t, defaultModDir)

						// We manually build bingo binary to make sure GOCACHE will not hit us.
						bingoPath := filepath.Join(g.tmpDir, bingoBin)
						buildInitialGobin(t, bingoPath)

						expectBingoListRows(t, bingoExpectedCompatibilityRows, g.ExecOutput(t, p.root, bingoPath, "list"))
						testutil.Equals(t, []string{}, g.existingBinaries(t))

						defer p.assertNotChanged(t, defaultModDir)

						if isGoProject {
							// This should work without cd even.
							_, err := execCmd(p.root, nil, "go", "build", "-mod=mod", "-modfile="+filepath.Join(defaultModDir, "buildable.1.mod"),
								"-o="+filepath.Join(g.gobin, "buildable-v1.0.0"), "github.com/bwplotka/bingo-testmodule/buildable")
							testutil.Ok(t, err)
							_, err = execCmd(p.root, nil, "go", "build", "-mod=mod", "-modfile="+filepath.Join(defaultModDir, "buildable-withReplace.mod"),
								"-o="+filepath.Join(g.gobin, "buildable-withReplace-v0.0.0-20221007091003-fe4d42a37d92"), "github.com/bwplotka/bingo-testmodule/buildable")
							testutil.Ok(t, err)
						} else {
							// For no go projects we have this "bug" that requires go.mod to be present.
							_, err := execCmd(filepath.Join(p.root, defaultModDir), nil, "go", "build", "-mod=mod", "-modfile=buildable.1.mod",
								"-o="+filepath.Join(g.gobin, "buildable-v1.0.0"), "github.com/bwplotka/bingo-testmodule/buildable")
							testutil.Ok(t, err)
							_, err = execCmd(filepath.Join(p.root, defaultModDir), nil, "go", "build", "-mod=mod", "-modfile=buildable-withReplace.mod",
								"-o="+filepath.Join(g.gobin, "buildable-withReplace-v0.0.0-20221007091003-fe4d42a37d92"), "github.com/bwplotka/bingo-testmodule/buildable")
							testutil.Ok(t, err)

						}
						testutil.Equals(t, "buildable\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "buildable-v1.0.0")))
						testutil.Equals(t, "buildable\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "buildable-withReplace-v0.0.0-20221007091003-fe4d42a37d92")))
						testutil.Equals(t, []string{"buildable-v1.0.0", "buildable-withReplace-v0.0.0-20221007091003-fe4d42a37d92"}, g.existingBinaries(t))
					})
					// TODO(bwplotka): Test variables.env as well.
					t.Run("Makefile", func(t *testing.T) {
						if !goVersion.LessThan(version.Go116) {
							// These projects are configured with modules but the generated Makefiles do not contain the
							// `-mod=mod` argument, and that makes those Makefiles incompatible with Go modules in 1.16.
							// Let's run bingo get to simulate
							for _, v := range []string{"v0_1_1", "v0_2_0", "v0_2_1", "v0_2_2"} {
								if strings.HasSuffix(dir, v) {
									t.Skipf("skipping %q in Go >= 1.16 because the generated Makefile is missing the '-mod-mod' flag and it is needed in Go >= 1.16", dir)
								}
							}
						}

						// Make is one of test requirement.
						makePath := makePath(t)

						g.Clear(t)

						// Copy testproject at the beginning to temp dir.
						prjRoot := filepath.Join(g.tmpDir, "testproject")
						p := newTestProject(t, dir, prjRoot, isGoProject)
						p.assertNotChanged(t, defaultModDir)

						// We manually build bingo binary to make sure GOCACHE will not hit us.
						bingoPath := filepath.Join(g.tmpDir, bingoBin)
						buildInitialGobin(t, bingoPath)

						expectBingoListRows(t, bingoExpectedCompatibilityRows, g.ExecOutput(t, p.root, bingoPath, "list"))
						testutil.Equals(t, []string{}, g.existingBinaries(t))

						g.ExecOutput(t, p.root, makePath, "buildable-v2-exists")
						testutil.Equals(t, []string{"buildable-v2-v2.0.0"}, g.existingBinaries(t))
						testutil.Equals(t, "buildable\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "buildable-v2-v2.0.0")))
						testutil.NotOk(t, g.ExpectErr(p.root, filepath.Join(g.gobin, "buildable-withReplace-v0.0.0-20221007091003-fe4d42a37d92")))

						testutil.Equals(t, "(re)installing "+g.gobin+"/buildable-withReplace-v0.0.0-20221007091003-fe4d42a37d92\nchecking buildable-with-replace\n", g.ExecOutput(t, p.root, makePath, "buildable-withreplace-exists"))
						testutil.Equals(t, "checking buildable-v2\n", g.ExecOutput(t, p.root, makePath, "buildable-v2-exists"))
						testutil.Equals(t, []string{"buildable-v2-v2.0.0", "buildable-withReplace-v0.0.0-20221007091003-fe4d42a37d92"}, g.existingBinaries(t))
						testutil.Equals(t, "buildable\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "buildable-v2-v2.0.0")))
						testutil.Equals(t, "buildable\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "buildable-withReplace-v0.0.0-20221007091003-fe4d42a37d92")))

						t.Run("Delete binary file, expect reinstall", func(t *testing.T) {
							_, err := execCmd(g.gobin, nil, "rm", "buildable-v2-v2.0.0")
							testutil.Ok(t, err)
							testutil.Equals(t, []string{"buildable-withReplace-v0.0.0-20221007091003-fe4d42a37d92"}, g.existingBinaries(t))

							testutil.Equals(t, "(re)installing "+g.gobin+"/buildable-v2-v2.0.0\nchecking buildable-v2\n", g.ExecOutput(t, p.root, makePath, "buildable-v2-exists"))
							testutil.Equals(t, "checking buildable-with-replace\n", g.ExecOutput(t, p.root, makePath, "buildable-withreplace-exists"))
							testutil.Equals(t, []string{"buildable-v2-v2.0.0", "buildable-withReplace-v0.0.0-20221007091003-fe4d42a37d92"}, g.existingBinaries(t))
						})
					})
				})
			}
		})
	}
}

func TestGet_ModuleCases(t *testing.T) {
	g := newIsolatedGoEnv(t, defaultGoProxy)
	defer g.Close(t)

	var goVersion *semver.Version
	{
		r, err := runner.NewRunner(context.Background(), nil, false, "go")
		testutil.Ok(t, err)
		goVersion = r.GoVersion()
	}

	t.Run("benchstat: latest in case where no major version is found", func(t *testing.T) {
		g.Clear(t)

		testutil.Ok(t, os.MkdirAll(filepath.Join(g.tmpDir, "newproject"), os.ModePerm))
		p := newTestProject(t, filepath.Join(g.tmpDir, "newproject"), filepath.Join(g.tmpDir, "testproject"), false)
		p.assertNotChanged(t)

		// We manually build bingo binary to make sure GOCACHE will not hit us.
		bingoPath := filepath.Join(g.tmpDir, bingoBin)
		buildInitialGobin(t, bingoPath)

		expectBingoListRows(t, []row(nil), g.ExecOutput(t, p.root, bingoPath, "list"))
		testutil.Equals(t, []string{}, g.existingBinaries(t))

		fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "golang.org/x/perf/cmd/benchstat@latest"))
	})

	t.Run("module with generics", func(t *testing.T) {
		g.Clear(t)

		testutil.Ok(t, os.MkdirAll(filepath.Join(g.tmpDir, "newproject"), os.ModePerm))
		p := newTestProject(t, filepath.Join(g.tmpDir, "newproject"), filepath.Join(g.tmpDir, "testproject"), false)
		p.assertNotChanged(t)

		// We manually build bingo binary to make sure GOCACHE will not hit us.
		bingoPath := filepath.Join(g.tmpDir, bingoBin)
		buildInitialGobin(t, bingoPath)

		expectBingoListRows(t, []row(nil), g.ExecOutput(t, p.root, bingoPath, "list"))
		testutil.Equals(t, []string{}, g.existingBinaries(t))

		if goVersion.LessThan(semver.MustParse("v1.18")) {
			err := g.ExpectErr(p.root, bingoPath, "get", "github.com/bwplotka/bingo-testmodule/buildable2@d48721795572f7b824f60a5b0623e524b263ed0c")
			testutil.NotOk(t, err)
		} else {
			fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "github.com/bwplotka/bingo-testmodule/buildable2@d48721795572f7b824f60a5b0623e524b263ed0c"))
		}
	})

	// Tricky cases TODO.

	// Regression against https://github.com/bwplotka/bingo/issues/125
	//t.Run("golangcilint", func(t *testing.T) {
	//	g.Clear(t)
	//
	//	testutil.Ok(t, os.MkdirAll(filepath.Join(g.tmpDir, "newproject"), os.ModePerm))
	//	p := newTestProject(t, filepath.Join(g.tmpDir, "newproject"), filepath.Join(g.tmpDir, "testproject"), true)
	//	p.assertNotChanged(t)
	//
	//	// We manually build bingo binary to make sure GOCACHE will not hit us.
	//	bingoPath := filepath.Join(g.tmpDir, bingoBin)
	//	buildInitialGobin(t, bingoPath)
	//
	//	expectBingoListRows(t, []row(nil), g.ExecOutput(t, p.root, bingoPath, "list"))
	//	testutil.Equals(t, []string{}, g.existingBinaries(t))
	//
	//	fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "github.com/golangci/golangci-lint/cmd/golangci-lint@v1.51.1"))
	//})

	//	// Regression test against https://github.com/bwplotka/bingo/issues/65.
	//	name: "get tool with capital letters in name",
	//	do: func(t *testing.T, g *goEnv, p *testProject) {
	// Println(g.ExecOutput(t, p.root, bingoPath, "get", "github.com/githubnemo/CompileDaemon@87e39427f4ba26da4400abf3b26b2e58bfc9ebe6"))

	// 	name: "Get tricky case with replace (thanos)",
	//			do: func(t *testing.T, g *goEnv, p *testProject) {
	//				// For Thanos/Prom/k8s etc without replace even go-get or list fails. This should be handled well.

	// Exclude fields.

	// 	{
	//			name: "Get tricky case with retract (ginkgo)",
	//			do: func(t *testing.T, g *goEnv, p *testProject) {
	//				if goVersion.LessThan(version.Go116) {
	//					t.Skip("Go version below 1.16 are not understanding go modules with retract directive; skip it.")
	//	fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "github.com/onsi/ginkgo/ginkgo@v1.16.4"))
	//				}

	// // TODO(bwplotka): Uncomment when https://github.com/githubnemo/CompileDaemon/pull/76 is merged or
	//		// https://github.com/bwplotka/bingo/issues/31
	//		//{
	//		//	// Regression test against https://github.com/bwplotka/bingo/issues/65.
	//		//	name: "get tool with capital letters in name (pre modules)",
	// fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "github.com/githubnemo/CompileDaemon@v1.2.1"))

	// 	name: "get istio.io/tools/cmd/cue-gen@355a0b7a6ba743d14e3a43a3069287086207f35c (short module base path)",
	//						do: func(t *testing.T) {
	//							fmt.Println(g.ExecOutput(t, p.root, bingoPath, "get", "istio.io/tools/cmd/cue-gen@355a0b7a6ba743d14e3a43a3069287086207f35c"))
	//						},
}

type row struct {
	name, binName, pkgVersion, buildEnvVars, buildFlags string
}

func removeTabDups(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	var last rune
	for i, r := range s {
		if r != last || r != '\t' || i == 0 {
			b.WriteRune(r)
			last = r
		}
	}
	return b.String()
}

func expectBingoListRows(t testing.TB, expect []row, output string) {
	t.Helper()

	var (
		trimmed = strings.TrimPrefix(removeTabDups(output), bingo.PackageRenderablesPrintHeader)
		got     []row
	)
	for _, line := range strings.Split(trimmed, "\n") {
		s := strings.Fields(line)
		if len(s) == 0 {
			break
		}
		r := row{name: s[0]}
		if len(s) > 1 {
			r.binName = s[1]
		}
		if len(s) > 2 {
			r.pkgVersion = s[2]
		}
		if len(s) > 3 {
			r.buildEnvVars = s[3]
		}
		if len(s) > 4 {
			r.buildFlags = s[4]
		}
		got = append(got, r)
	}
	testutil.Equals(t, expect, got)
}

func TestExpectBingoListRows(t *testing.T) {
	expectBingoListRows(t, []row{
		{name: "copyright", binName: "copyright-v0.0.0-20210112004814-138d5e5695fe", pkgVersion: "github.com/efficientgo/tools/copyright@v0.0.0-20210112004814-138d5e5695fe"},
		{name: "embedmd", binName: "embedmd-v1.0.0", pkgVersion: "github.com/campoy/embedmd@v1.0.0", buildEnvVars: "CGO_ENABLED=1", buildFlags: "-tags=lol"},
		{name: "faillint", binName: "faillint-v1.5.0", pkgVersion: "github.com/fatih/faillint@v1.5.0"},
		{name: "goimports", binName: "goimports-v0.0.0-20210112230658-8b4aab62c064", pkgVersion: "golang.org/x/tools/cmd/goimports@v0.0.0-20210112230658-8b4aab62c064"},
		{name: "golangci-lint", binName: "golangci-lint-v1.26.0", pkgVersion: "github.com/golangci/golangci-lint/cmd/golangci-lint@v1.26.0"},
		{name: "mdox", binName: "mdox-v0.2.1", pkgVersion: "github.com/bwplotka/mdox@v0.2.1"},
		{name: "misspell", binName: "misspell-v0.3.4", pkgVersion: "github.com/client9/misspell/cmd/misspell@v0.3.4"},
		{name: "proxy", binName: "proxy-v0.10.0", pkgVersion: "github.com/gomods/athens/cmd/proxy@v0.10.0"},
	}, `Name		Binary Name					Package @ Version								Build EnvVars	Build Flags
----		-----------					-----------------								-------------	-----------
copyright	copyright-v0.0.0-20210112004814-138d5e5695fe	github.com/efficientgo/tools/copyright@v0.0.0-20210112004814-138d5e5695fe			
embedmd		embedmd-v1.0.0					github.com/campoy/embedmd@v1.0.0						CGO_ENABLED=1	-tags=lol
faillint	faillint-v1.5.0					github.com/fatih/faillint@v1.5.0								
goimports	goimports-v0.0.0-20210112230658-8b4aab62c064	golang.org/x/tools/cmd/goimports@v0.0.0-20210112230658-8b4aab62c064				
golangci-lint	golangci-lint-v1.26.0				github.com/golangci/golangci-lint/cmd/golangci-lint@v1.26.0					
mdox		mdox-v0.2.1					github.com/bwplotka/mdox@v0.2.1									
misspell	misspell-v0.3.4					github.com/client9/misspell/cmd/misspell@v0.3.4							
proxy		proxy-v0.10.0					github.com/gomods/athens/cmd/proxy@v0.10.0
`)
}
