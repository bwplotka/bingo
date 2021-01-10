// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package main_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwplotka/bingo/pkg/version"
	"github.com/efficientgo/tools/core/pkg/testutil"
)

const (
	bingoBin      = "bingo"
	defaultModDir = ".bingo"
)

const defaultGoProxy = "https://proxy.golang.org"

// TODO(bwplotka): Test running versions. To do so we might want to setup small binary printing Version at each commit.
// $GOBIN has to be set for this test to run properly.
func TestGet(t *testing.T) {
	currTestCaseDir := fmt.Sprintf("testdata/testproject_with_bingo_%s", strings.ReplaceAll(version.Version, ".", "_"))

	g := newIsolatedGoEnv(t, defaultGoProxy)
	defer g.Close(t)

	if ok := t.Run("empty project with advanced cases", func(t *testing.T) {
		for _, isGoProject := range []bool{false, true} {
			if ok := t.Run(fmt.Sprintf("isGoProject=%v", isGoProject), func(t *testing.T) {
				g.Clear(t)

				// We manually build bingo binary to make sure GOCACHE will not hit us.
				goBinPath := filepath.Join(g.tmpDir, bingoBin)
				buildInitialGobin(t, goBinPath)

				testutil.Ok(t, os.MkdirAll(filepath.Join(g.tmpDir, "newproject"), os.ModePerm))
				p := newTestProject(t, filepath.Join(g.tmpDir, "newproject"), filepath.Join(g.tmpDir, "testproject"), isGoProject)
				p.assertNotChanged(t)

				for _, tcase := range []struct {
					name string
					do   func(t *testing.T)

					expectBinaries []string
					expectRows     []row
				}{
					{
						name: "get github.com/fatih/faillint@v1.4.0",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "github.com/fatih/faillint@v1.4.0"))
							testutil.Equals(t, g.ExecOutput(t, p.root, goBinPath, "list", "faillint"), g.ExecOutput(t, p.root, goBinPath, "list"))
						},
						expectRows:     []row{{name: "faillint", binName: "faillint-v1.4.0", pkgVersion: "github.com/fatih/faillint@v1.4.0"}},
						expectBinaries: []string{"faillint-v1.4.0"},
					},
					{
						name: "get github.com/bwplotka/bingo/testdata/module/buildable@2e6391144e85de14181f8e47b77d64b94a7ca3a8",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "github.com/bwplotka/bingo/testdata/module/buildable@2e6391144e85de14181f8e47b77d64b94a7ca3a8"))
							// Check if installed tool is what we expect.
							testutil.Equals(t, "module.buildable 2\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "buildable-v0.0.0-20210109093942-2e6391144e85")))
						},
						expectRows: []row{
							{name: "buildable", binName: "buildable-v0.0.0-20210109093942-2e6391144e85", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109093942-2e6391144e85"},
							{name: "faillint", binName: "faillint-v1.4.0", pkgVersion: "github.com/fatih/faillint@v1.4.0"},
						},
						expectBinaries: []string{"buildable-v0.0.0-20210109093942-2e6391144e85", "faillint-v1.4.0"},
					},
					{
						name: "get same tool; should be noop",
						do: func(t *testing.T) {
							// TODO(bwplotka): Assert if actually noop.
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "github.com/bwplotka/bingo/testdata/module/buildable@2e6391144e85de14181f8e47b77d64b94a7ca3a8"))
						},
						expectRows: []row{
							{name: "buildable", binName: "buildable-v0.0.0-20210109093942-2e6391144e85", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109093942-2e6391144e85"},
							{name: "faillint", binName: "faillint-v1.4.0", pkgVersion: "github.com/fatih/faillint@v1.4.0"},
						},
						expectBinaries: []string{"buildable-v0.0.0-20210109093942-2e6391144e85", "faillint-v1.4.0"},
					},
					{
						name: "get github.com/bwplotka/bingo/testdata/module/buildable@375d0606849d58d106888f5c5ed80887eb899686 (update by path)",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "github.com/bwplotka/bingo/testdata/module/buildable@375d0606849d58d106888f5c5ed80887eb899686"))

							// Check if installed tool is what we expect.
							testutil.Equals(t, "module.buildable 2\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "buildable-v0.0.0-20210109093942-2e6391144e85")))
							testutil.Equals(t, "module.buildable 2.1\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "buildable-v0.0.0-20210109094001-375d0606849d")))
						},
						expectRows: []row{
							{name: "buildable", binName: "buildable-v0.0.0-20210109094001-375d0606849d", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109094001-375d0606849d"},
							{name: "faillint", binName: "faillint-v1.4.0", pkgVersion: "github.com/fatih/faillint@v1.4.0"},
						},
						expectBinaries: []string{"buildable-v0.0.0-20210109093942-2e6391144e85", "buildable-v0.0.0-20210109094001-375d0606849d", "faillint-v1.4.0"},
					},
					{
						name: "get faillint@v1.5.0 (update by name)",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "faillint@v1.5.0"))
						},
						expectRows: []row{
							{name: "buildable", binName: "buildable-v0.0.0-20210109094001-375d0606849d", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109094001-375d0606849d"},
							{name: "faillint", binName: "faillint-v1.5.0", pkgVersion: "github.com/fatih/faillint@v1.5.0"},
						},
						expectBinaries: []string{"buildable-v0.0.0-20210109093942-2e6391144e85", "buildable-v0.0.0-20210109094001-375d0606849d", "faillint-v1.4.0", "faillint-v1.5.0"},
					},
					{
						name: "get faillint@v1.3.0 (downgrade by name)",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "faillint@v1.3.0"))
						},
						expectRows: []row{
							{name: "buildable", binName: "buildable-v0.0.0-20210109094001-375d0606849d", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109094001-375d0606849d"},
							{name: "faillint", binName: "faillint-v1.3.0", pkgVersion: "github.com/fatih/faillint@v1.3.0"},
						},
						expectBinaries: []string{"buildable-v0.0.0-20210109093942-2e6391144e85", "buildable-v0.0.0-20210109094001-375d0606849d", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0"},
					},
					{
						name: "get -n=buildable_old github.com/bwplotka/bingo/testdata/module/buildable@375d0606849d58d106888f5c5ed80887eb899686 (get buildable from same module under different name)",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "-n=buildable_old", "github.com/bwplotka/bingo/testdata/module/buildable@375d0606849d58d106888f5c5ed80887eb899686"))

							// Check if installed tool is what we expect.
							testutil.Equals(t, "module.buildable 2.1\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "buildable_old-v0.0.0-20210109094001-375d0606849d")))
							testutil.Equals(t, "module.buildable 2\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "buildable-v0.0.0-20210109093942-2e6391144e85")))
							testutil.Equals(t, "module.buildable 2.1\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "buildable-v0.0.0-20210109094001-375d0606849d")))
						},
						expectRows: []row{
							{name: "buildable", binName: "buildable-v0.0.0-20210109094001-375d0606849d", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109094001-375d0606849d"},
							{name: "buildable_old", binName: "buildable_old-v0.0.0-20210109094001-375d0606849d", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109094001-375d0606849d"},
							{name: "faillint", binName: "faillint-v1.3.0", pkgVersion: "github.com/fatih/faillint@v1.3.0"},
						},
						expectBinaries: []string{
							"buildable-v0.0.0-20210109093942-2e6391144e85", "buildable-v0.0.0-20210109094001-375d0606849d",
							"buildable_old-v0.0.0-20210109094001-375d0606849d",
							"faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0",
						},
					},
					{
						name: "get buildable_old@2e6391144e85de14181f8e47b77d64b94a7ca3a8 (downgrade buildable from same module under different name)",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "buildable_old@2e6391144e85de14181f8e47b77d64b94a7ca3a8"))

							// Check if installed tool is what we expect.
							testutil.Equals(t, "module.buildable 2\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "buildable_old-v0.0.0-20210109093942-2e6391144e85")))
							testutil.Equals(t, "module.buildable 2.1\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "buildable_old-v0.0.0-20210109094001-375d0606849d")))
							testutil.Equals(t, "module.buildable 2\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "buildable-v0.0.0-20210109093942-2e6391144e85")))
							testutil.Equals(t, "module.buildable 2.1\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "buildable-v0.0.0-20210109094001-375d0606849d")))
						},
						expectRows: []row{
							{name: "buildable", binName: "buildable-v0.0.0-20210109094001-375d0606849d", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109094001-375d0606849d"},
							{name: "buildable_old", binName: "buildable_old-v0.0.0-20210109093942-2e6391144e85", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109093942-2e6391144e85"},
							{name: "faillint", binName: "faillint-v1.3.0", pkgVersion: "github.com/fatih/faillint@v1.3.0"},
						},
						expectBinaries: []string{
							"buildable-v0.0.0-20210109093942-2e6391144e85", "buildable-v0.0.0-20210109094001-375d0606849d",
							"buildable_old-v0.0.0-20210109093942-2e6391144e85", "buildable_old-v0.0.0-20210109094001-375d0606849d",
							"faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0",
						},
					},
					{
						name: "get github.com/go-bindata/go-bindata/go-bindata@v3.1.1 (pre go module project)",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "github.com/go-bindata/go-bindata/go-bindata@v3.1.1"))
						},
						expectRows: []row{
							{name: "buildable", binName: "buildable-v0.0.0-20210109094001-375d0606849d", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109094001-375d0606849d"},
							{name: "buildable_old", binName: "buildable_old-v0.0.0-20210109093942-2e6391144e85", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109093942-2e6391144e85"},
							{name: "faillint", binName: "faillint-v1.3.0", pkgVersion: "github.com/fatih/faillint@v1.3.0"},
							{name: "go-bindata", binName: "go-bindata-v3.1.1+incompatible", pkgVersion: "github.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible"},
						},
						expectBinaries: []string{
							"buildable-v0.0.0-20210109093942-2e6391144e85", "buildable-v0.0.0-20210109094001-375d0606849d",
							"buildable_old-v0.0.0-20210109093942-2e6391144e85", "buildable_old-v0.0.0-20210109094001-375d0606849d",
							"faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0",
							"go-bindata-v3.1.1+incompatible",
						},
					},
					{
						name: "get github.com/bwplotka/bingo/testdata/module/buildable2@2e6391144e85de14181f8e47b77d64b94a7ca3a8 (get buildable2 from same module from different version!)",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "github.com/bwplotka/bingo/testdata/module/buildable2@2e6391144e85de14181f8e47b77d64b94a7ca3a8"))

							// Check if installed tool is what we expect.
							testutil.Equals(t, "module.buildable 2.1\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "buildable_old-v0.0.0-20210109094001-375d0606849d")))
							testutil.Equals(t, "module.buildable 2\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "buildable-v0.0.0-20210109093942-2e6391144e85")))
							testutil.Equals(t, "module.buildable 2.1\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "buildable-v0.0.0-20210109094001-375d0606849d")))
							testutil.Equals(t, "module.buildable2 2\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "buildable2-v0.0.0-20210109093942-2e6391144e85")))
						},
						expectRows: []row{
							{name: "buildable", binName: "buildable-v0.0.0-20210109094001-375d0606849d", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109094001-375d0606849d"},
							{name: "buildable2", binName: "buildable2-v0.0.0-20210109093942-2e6391144e85", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable2@v0.0.0-20210109093942-2e6391144e85"},
							{name: "buildable_old", binName: "buildable_old-v0.0.0-20210109093942-2e6391144e85", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109093942-2e6391144e85"},
							{name: "faillint", binName: "faillint-v1.3.0", pkgVersion: "github.com/fatih/faillint@v1.3.0"},
							{name: "go-bindata", binName: "go-bindata-v3.1.1+incompatible", pkgVersion: "github.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible"},
						},
						expectBinaries: []string{
							"buildable-v0.0.0-20210109093942-2e6391144e85", "buildable-v0.0.0-20210109094001-375d0606849d", "buildable2-v0.0.0-20210109093942-2e6391144e85",
							"buildable_old-v0.0.0-20210109093942-2e6391144e85", "buildable_old-v0.0.0-20210109094001-375d0606849d",
							"faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0",
							"go-bindata-v3.1.1+incompatible",
						},
					},
					{
						name: "get -n=wr_buildable github.com/bwplotka/bingo/testdata/module_with_replace/buildable@ab990d1be30bcbad4d35220e0c98e8f57289f113 (get buildable from same module with relevant replaces)",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "-n=wr_buildable", "github.com/bwplotka/bingo/testdata/module_with_replace/buildable@ab990d1be30bcbad4d35220e0c98e8f57289f113"))

							// Check if installed tool is what we expect.
							testutil.Equals(t, "module_with_replace.buildable 2.8\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "wr_buildable-v0.0.0-20210110214650-ab990d1be30b")))
						},
						expectRows: []row{
							{name: "buildable", binName: "buildable-v0.0.0-20210109094001-375d0606849d", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109094001-375d0606849d"},
							{name: "buildable2", binName: "buildable2-v0.0.0-20210109093942-2e6391144e85", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable2@v0.0.0-20210109093942-2e6391144e85"},
							{name: "buildable_old", binName: "buildable_old-v0.0.0-20210109093942-2e6391144e85", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109093942-2e6391144e85"},
							{name: "faillint", binName: "faillint-v1.3.0", pkgVersion: "github.com/fatih/faillint@v1.3.0"},
							{name: "go-bindata", binName: "go-bindata-v3.1.1+incompatible", pkgVersion: "github.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible"},
							{name: "wr_buildable", binName: "wr_buildable-v0.0.0-20210110214650-ab990d1be30b", pkgVersion: "github.com/bwplotka/bingo/testdata/module_with_replace/buildable@v0.0.0-20210110214650-ab990d1be30b"},
						},
						expectBinaries: []string{
							"buildable-v0.0.0-20210109093942-2e6391144e85", "buildable-v0.0.0-20210109094001-375d0606849d", "buildable2-v0.0.0-20210109093942-2e6391144e85",
							"buildable_old-v0.0.0-20210109093942-2e6391144e85", "buildable_old-v0.0.0-20210109094001-375d0606849d",
							"faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0",
							"go-bindata-v3.1.1+incompatible",
							"wr_buildable-v0.0.0-20210110214650-ab990d1be30b",
						},
					},
					{
						name: "get wr_buildable@ccbd4039b94aac79d926ba5eebfe6a132a728ed8 (dowgrade buildable with different replaces - trickier than you think!)",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "wr_buildable@ccbd4039b94aac79d926ba5eebfe6a132a728ed8"))

							// Check if installed tool is what we expect.
							testutil.Equals(t, "module_with_replace.buildable 2.8\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "wr_buildable-v0.0.0-20210110214650-ab990d1be30b")))
							testutil.Equals(t, "module_with_replace.buildable 2.7\n", g.ExecOutput(t, p.root, filepath.Join(g.gobin, "wr_buildable-v0.0.0-20210109165512-ccbd4039b94a")))
						},
						expectRows: []row{
							{name: "buildable", binName: "buildable-v0.0.0-20210109094001-375d0606849d", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109094001-375d0606849d"},
							{name: "buildable2", binName: "buildable2-v0.0.0-20210109093942-2e6391144e85", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable2@v0.0.0-20210109093942-2e6391144e85"},
							{name: "buildable_old", binName: "buildable_old-v0.0.0-20210109093942-2e6391144e85", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109093942-2e6391144e85"},
							{name: "faillint", binName: "faillint-v1.3.0", pkgVersion: "github.com/fatih/faillint@v1.3.0"},
							{name: "go-bindata", binName: "go-bindata-v3.1.1+incompatible", pkgVersion: "github.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible"},
							{name: "wr_buildable", binName: "wr_buildable-v0.0.0-20210109165512-ccbd4039b94a", pkgVersion: "github.com/bwplotka/bingo/testdata/module_with_replace/buildable@v0.0.0-20210109165512-ccbd4039b94a"},
						},
						expectBinaries: []string{
							"buildable-v0.0.0-20210109093942-2e6391144e85", "buildable-v0.0.0-20210109094001-375d0606849d", "buildable2-v0.0.0-20210109093942-2e6391144e85",
							"buildable_old-v0.0.0-20210109093942-2e6391144e85", "buildable_old-v0.0.0-20210109094001-375d0606849d",
							"faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0",
							"go-bindata-v3.1.1+incompatible",
							"wr_buildable-v0.0.0-20210109165512-ccbd4039b94a", "wr_buildable-v0.0.0-20210110214650-ab990d1be30b",
						},
					},
					{
						name: "naive installing package that would result with `cmd` name fails - different name is suggested",
						do: func(t *testing.T) {
							testutil.NotOk(t, g.ExpectErr(p.root, goBinPath, "get", "github.com/bwplotka/promeval@v0.3.0"))
						},
						expectRows: []row{
							{name: "buildable", binName: "buildable-v0.0.0-20210109094001-375d0606849d", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109094001-375d0606849d"},
							{name: "buildable2", binName: "buildable2-v0.0.0-20210109093942-2e6391144e85", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable2@v0.0.0-20210109093942-2e6391144e85"},
							{name: "buildable_old", binName: "buildable_old-v0.0.0-20210109093942-2e6391144e85", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109093942-2e6391144e85"},
							{name: "faillint", binName: "faillint-v1.3.0", pkgVersion: "github.com/fatih/faillint@v1.3.0"},
							{name: "go-bindata", binName: "go-bindata-v3.1.1+incompatible", pkgVersion: "github.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible"},
							{name: "wr_buildable", binName: "wr_buildable-v0.0.0-20210109165512-ccbd4039b94a", pkgVersion: "github.com/bwplotka/bingo/testdata/module_with_replace/buildable@v0.0.0-20210109165512-ccbd4039b94a"},
						},
						expectBinaries: []string{
							"buildable-v0.0.0-20210109093942-2e6391144e85", "buildable-v0.0.0-20210109094001-375d0606849d", "buildable2-v0.0.0-20210109093942-2e6391144e85",
							"buildable_old-v0.0.0-20210109093942-2e6391144e85", "buildable_old-v0.0.0-20210109094001-375d0606849d",
							"faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0",
							"go-bindata-v3.1.1+incompatible",
							"wr_buildable-v0.0.0-20210109165512-ccbd4039b94a", "wr_buildable-v0.0.0-20210110214650-ab990d1be30b",
						},
					},
					{
						name: "install package with go name should fail. (this is due to clash with go.mod)",
						do: func(t *testing.T) {
							testutil.NotOk(t, g.ExpectErr(p.root, goBinPath, "get", "github.com/something/go"))
						},
						expectRows: []row{
							{name: "buildable", binName: "buildable-v0.0.0-20210109094001-375d0606849d", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109094001-375d0606849d"},
							{name: "buildable2", binName: "buildable2-v0.0.0-20210109093942-2e6391144e85", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable2@v0.0.0-20210109093942-2e6391144e85"},
							{name: "buildable_old", binName: "buildable_old-v0.0.0-20210109093942-2e6391144e85", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109093942-2e6391144e85"},
							{name: "faillint", binName: "faillint-v1.3.0", pkgVersion: "github.com/fatih/faillint@v1.3.0"},
							{name: "go-bindata", binName: "go-bindata-v3.1.1+incompatible", pkgVersion: "github.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible"},
							{name: "wr_buildable", binName: "wr_buildable-v0.0.0-20210109165512-ccbd4039b94a", pkgVersion: "github.com/bwplotka/bingo/testdata/module_with_replace/buildable@v0.0.0-20210109165512-ccbd4039b94a"},
						},
						expectBinaries: []string{
							"buildable-v0.0.0-20210109093942-2e6391144e85", "buildable-v0.0.0-20210109094001-375d0606849d", "buildable2-v0.0.0-20210109093942-2e6391144e85",
							"buildable_old-v0.0.0-20210109093942-2e6391144e85", "buildable_old-v0.0.0-20210109094001-375d0606849d",
							"faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0",
							"go-bindata-v3.1.1+incompatible",
							"wr_buildable-v0.0.0-20210109165512-ccbd4039b94a", "wr_buildable-v0.0.0-20210110214650-ab990d1be30b",
						},
					},
					{
						name: "Get array of 4 versions of faillint under f2 name",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "-n", "f2", "github.com/fatih/faillint@v1.5.0,v1.1.0,v1.2.0,v1.0.0"))
						},
						expectRows: []row{
							{name: "buildable", binName: "buildable-v0.0.0-20210109094001-375d0606849d", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109094001-375d0606849d"},
							{name: "buildable2", binName: "buildable2-v0.0.0-20210109093942-2e6391144e85", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable2@v0.0.0-20210109093942-2e6391144e85"},
							{name: "buildable_old", binName: "buildable_old-v0.0.0-20210109093942-2e6391144e85", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109093942-2e6391144e85"},
							{name: "f2", binName: "f2-v1.5.0", pkgVersion: "github.com/fatih/faillint@v1.5.0"},
							{name: "f2", binName: "f2-v1.1.0", pkgVersion: "github.com/fatih/faillint@v1.1.0"},
							{name: "f2", binName: "f2-v1.2.0", pkgVersion: "github.com/fatih/faillint@v1.2.0"},
							{name: "f2", binName: "f2-v1.0.0", pkgVersion: "github.com/fatih/faillint@v1.0.0"},
							{name: "faillint", binName: "faillint-v1.3.0", pkgVersion: "github.com/fatih/faillint@v1.3.0"},
							{name: "go-bindata", binName: "go-bindata-v3.1.1+incompatible", pkgVersion: "github.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible"},
							{name: "wr_buildable", binName: "wr_buildable-v0.0.0-20210109165512-ccbd4039b94a", pkgVersion: "github.com/bwplotka/bingo/testdata/module_with_replace/buildable@v0.0.0-20210109165512-ccbd4039b94a"},
						},
						expectBinaries: []string{
							"buildable-v0.0.0-20210109093942-2e6391144e85", "buildable-v0.0.0-20210109094001-375d0606849d", "buildable2-v0.0.0-20210109093942-2e6391144e85",
							"buildable_old-v0.0.0-20210109093942-2e6391144e85", "buildable_old-v0.0.0-20210109094001-375d0606849d",
							"f2-v1.5.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.0.0",
							"faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0",
							"go-bindata-v3.1.1+incompatible",
							"wr_buildable-v0.0.0-20210109165512-ccbd4039b94a", "wr_buildable-v0.0.0-20210110214650-ab990d1be30b",
						},
					},
					{
						name: "<Special> Persist current state, to use for compatibility testing.",
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
						expectRows: []row{
							{name: "buildable", binName: "buildable-v0.0.0-20210109094001-375d0606849d", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109094001-375d0606849d"},
							{name: "buildable_old", binName: "buildable_old-v0.0.0-20210109093942-2e6391144e85", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109093942-2e6391144e85"},
							{name: "f2", binName: "f2-v1.5.0", pkgVersion: "github.com/fatih/faillint@v1.5.0"},
							{name: "f2", binName: "f2-v1.1.0", pkgVersion: "github.com/fatih/faillint@v1.1.0"},
							{name: "f2", binName: "f2-v1.2.0", pkgVersion: "github.com/fatih/faillint@v1.2.0"},
							{name: "f2", binName: "f2-v1.0.0", pkgVersion: "github.com/fatih/faillint@v1.0.0"},
							{name: "faillint", binName: "faillint-v1.3.0", pkgVersion: "github.com/fatih/faillint@v1.3.0"},
							{name: "go-bindata", binName: "go-bindata-v3.1.1+incompatible", pkgVersion: "github.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible"},
							{name: "wr_buildable", binName: "wr_buildable-v0.0.0-20210109165512-ccbd4039b94a", pkgVersion: "github.com/bwplotka/bingo/testdata/module_with_replace/buildable@v0.0.0-20210109165512-ccbd4039b94a"},
						},
						expectBinaries: []string{
							"buildable-v0.0.0-20210109093942-2e6391144e85", "buildable-v0.0.0-20210109094001-375d0606849d", "buildable2-v0.0.0-20210109093942-2e6391144e85",
							"buildable_old-v0.0.0-20210109093942-2e6391144e85", "buildable_old-v0.0.0-20210109094001-375d0606849d",
							"f2-v1.5.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.0.0",
							"faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0",
							"go-bindata-v3.1.1+incompatible",
							"wr_buildable-v0.0.0-20210109165512-ccbd4039b94a", "wr_buildable-v0.0.0-20210110214650-ab990d1be30b",
						},
					},
					{
						name: "Get array of 2 versions of normal faillint, despite being non array before, should work",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "faillint@v1.1.0,v1.0.0"))
						},
						expectRows: []row{
							{name: "buildable", binName: "buildable-v0.0.0-20210109094001-375d0606849d", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109094001-375d0606849d"},
							{name: "buildable_old", binName: "buildable_old-v0.0.0-20210109093942-2e6391144e85", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109093942-2e6391144e85"},
							{name: "f2", binName: "f2-v1.5.0", pkgVersion: "github.com/fatih/faillint@v1.5.0"},
							{name: "f2", binName: "f2-v1.1.0", pkgVersion: "github.com/fatih/faillint@v1.1.0"},
							{name: "f2", binName: "f2-v1.2.0", pkgVersion: "github.com/fatih/faillint@v1.2.0"},
							{name: "f2", binName: "f2-v1.0.0", pkgVersion: "github.com/fatih/faillint@v1.0.0"},
							{name: "faillint", binName: "faillint-v1.1.0", pkgVersion: "github.com/fatih/faillint@v1.1.0"},
							{name: "faillint", binName: "faillint-v1.0.0", pkgVersion: "github.com/fatih/faillint@v1.0.0"},
							{name: "go-bindata", binName: "go-bindata-v3.1.1+incompatible", pkgVersion: "github.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible"},
							{name: "wr_buildable", binName: "wr_buildable-v0.0.0-20210109165512-ccbd4039b94a", pkgVersion: "github.com/bwplotka/bingo/testdata/module_with_replace/buildable@v0.0.0-20210109165512-ccbd4039b94a"},
						},
						expectBinaries: []string{
							"buildable-v0.0.0-20210109093942-2e6391144e85", "buildable-v0.0.0-20210109094001-375d0606849d", "buildable2-v0.0.0-20210109093942-2e6391144e85",
							"buildable_old-v0.0.0-20210109093942-2e6391144e85", "buildable_old-v0.0.0-20210109094001-375d0606849d",
							"f2-v1.5.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.0.0",
							"faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0",
							"go-bindata-v3.1.1+incompatible",
							"wr_buildable-v0.0.0-20210109165512-ccbd4039b94a", "wr_buildable-v0.0.0-20210110214650-ab990d1be30b",
						},
					},
					{
						name: "Updating f2 to different version should work",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "f2@v1.3.0,v1.4.0"))
						},
						expectRows: []row{
							{name: "buildable", binName: "buildable-v0.0.0-20210109094001-375d0606849d", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109094001-375d0606849d"},
							{name: "buildable_old", binName: "buildable_old-v0.0.0-20210109093942-2e6391144e85", pkgVersion: "github.com/bwplotka/bingo/testdata/module/buildable@v0.0.0-20210109093942-2e6391144e85"},
							{name: "f2", binName: "f2-v1.3.0", pkgVersion: "github.com/fatih/faillint@v1.3.0"},
							{name: "f2", binName: "f2-v1.4.0", pkgVersion: "github.com/fatih/faillint@v1.4.0"},
							{name: "faillint", binName: "faillint-v1.1.0", pkgVersion: "github.com/fatih/faillint@v1.1.0"},
							{name: "faillint", binName: "faillint-v1.0.0", pkgVersion: "github.com/fatih/faillint@v1.0.0"},
							{name: "go-bindata", binName: "go-bindata-v3.1.1+incompatible", pkgVersion: "github.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible"},
							{name: "wr_buildable", binName: "wr_buildable-v0.0.0-20210109165512-ccbd4039b94a", pkgVersion: "github.com/bwplotka/bingo/testdata/module_with_replace/buildable@v0.0.0-20210109165512-ccbd4039b94a"},
						},
						expectBinaries: []string{
							"buildable-v0.0.0-20210109093942-2e6391144e85", "buildable-v0.0.0-20210109094001-375d0606849d", "buildable2-v0.0.0-20210109093942-2e6391144e85",
							"buildable_old-v0.0.0-20210109093942-2e6391144e85", "buildable_old-v0.0.0-20210109094001-375d0606849d",
							"f2-v1.5.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.0.0",
							"faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0",
							"go-bindata-v3.1.1+incompatible",
							"wr_buildable-v0.0.0-20210109165512-ccbd4039b94a", "wr_buildable-v0.0.0-20210110214650-ab990d1be30b",
						},
					},
					/// TODO: Rename to existing mod.
					{
						name: "Updating f2 to same multiple versions should fail",
						do: func(t *testing.T) {
							testutil.NotOk(t, g.ExpectErr(p.root, goBinPath, "get", "f2@v1.1.0,v1.4.0,v1.1.0"))
						},
						expectBinaries: []string{"f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Creating not existing foo to f3 should fail",
						do: func(t *testing.T) {
							testutil.NotOk(t, g.ExpectErr(p.root, goBinPath, "get", "-n", "f3", "x"))
						},
						expectBinaries: []string{"f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Renaming not existing foo to f3 should fail",
						do: func(t *testing.T) {
							testutil.NotOk(t, g.ExpectErr(p.root, goBinPath, "get", "-r", "f3", "x"))
						},
						expectBinaries: []string{"f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Cloning f2 to f2-clone should work",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "-n", "f2-clone", "f2"))
							testutil.Equals(t, "Name\t\t\tBinary Name\t\t\t\t\t\t\t\tPackage @ Version\t\t\t\t\t\t\t\t\t\t\t\n----\t\t\t-----------\t\t\t\t\t\t\t\t-----------------\t\t\t\t\t\t\t\t\t\t\t\nf2-clone\t\tf2-clone-v1.3.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.3.0\t\t\t\t\t\t\t\t\nf2-clone\t\tf2-clone-v1.4.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.4.0\t\t\t\t\t\t\t\t\nf2\t\t\tf2-v1.3.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.3.0\t\t\t\t\t\t\t\t\nf2\t\t\tf2-v1.4.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.4.0\t\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.1.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.1.0\t\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.0.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.0.0\t\t\t\t\t\t\t\t\ngo-bindata\tgo-bindata-v3.1.1+incompatible\t\t\t\tgithub.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible\t\t\ngoimports\t\tgoimports-v0.0.0-20200522201501-cb1345f3a375\t\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\t\ngoimports2\tgoimports2-v0.0.0-20200519175826-7521f6f42533\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200519175826-7521f6f42533", g.ExecOutput(t, p.root, goBinPath, "list"))
						},
						expectBinaries: []string{"f2-clone-v1.3.0", "f2-clone-v1.4.0", "f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					// check module install Latest
					{
						name: "Deleting f2-clone",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "f2-clone@none"))
							testutil.Equals(t, "Name\t\t\tBinary Name\t\t\t\t\t\t\t\tPackage @ Version\t\t\t\t\t\t\t\t\t\t\t\n----\t\t\t-----------\t\t\t\t\t\t\t\t-----------------\t\t\t\t\t\t\t\t\t\t\t\nf2\t\t\tf2-v1.3.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.3.0\t\t\t\t\t\t\t\t\nf2\t\t\tf2-v1.4.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.4.0\t\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.1.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.1.0\t\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.0.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.0.0\t\t\t\t\t\t\t\t\ngo-bindata\tgo-bindata-v3.1.1+incompatible\t\t\t\tgithub.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible\t\t\ngoimports\t\tgoimports-v0.0.0-20200522201501-cb1345f3a375\t\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\t\ngoimports2\tgoimports2-v0.0.0-20200519175826-7521f6f42533\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200519175826-7521f6f42533", g.ExecOutput(t, p.root, goBinPath, "list"))
						},
						expectBinaries: []string{"f2-clone-v1.3.0", "f2-clone-v1.4.0", "f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Renaming f2 to f3 should work",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "-r", "f3", "f2"))
							testutil.Equals(t, "Name\t\t\tBinary Name\t\t\t\t\t\t\t\tPackage @ Version\t\t\t\t\t\t\t\t\t\t\t\n----\t\t\t-----------\t\t\t\t\t\t\t\t-----------------\t\t\t\t\t\t\t\t\t\t\t\nf3\t\t\tf3-v1.3.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.3.0\t\t\t\t\t\t\t\t\nf3\t\t\tf3-v1.4.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.4.0\t\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.1.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.1.0\t\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.0.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.0.0\t\t\t\t\t\t\t\t\ngo-bindata\tgo-bindata-v3.1.1+incompatible\t\t\t\tgithub.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible\t\t\ngoimports\t\tgoimports-v0.0.0-20200522201501-cb1345f3a375\t\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\t\ngoimports2\tgoimports2-v0.0.0-20200519175826-7521f6f42533\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200519175826-7521f6f42533", g.ExecOutput(t, p.root, goBinPath, "list"))
						},
						expectBinaries: []string{"f2-clone-v1.3.0", "f2-clone-v1.4.0", "f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "f3-v1.3.0", "f3-v1.4.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Renaming f3 to f4 with certain version should work",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "-r", "f4", "f3@v1.1.0,v1.0.0"))
							testutil.Equals(t, "Name\t\t\tBinary Name\t\t\t\t\t\t\t\tPackage @ Version\t\t\t\t\t\t\t\t\t\t\t\n----\t\t\t-----------\t\t\t\t\t\t\t\t-----------------\t\t\t\t\t\t\t\t\t\t\t\nf4\t\t\tf4-v1.1.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.1.0\t\t\t\t\t\t\t\t\nf4\t\t\tf4-v1.0.0\t\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.0.0\t\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.1.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.1.0\t\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.0.0\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.0.0\t\t\t\t\t\t\t\t\ngo-bindata\tgo-bindata-v3.1.1+incompatible\t\t\t\tgithub.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible\t\t\ngoimports\t\tgoimports-v0.0.0-20200522201501-cb1345f3a375\t\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375\t\ngoimports2\tgoimports2-v0.0.0-20200519175826-7521f6f42533\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200519175826-7521f6f42533", g.ExecOutput(t, p.root, goBinPath, "list"))
						},
						expectBinaries: []string{"f2-clone-v1.3.0", "f2-clone-v1.4.0", "f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "f3-v1.3.0", "f3-v1.4.0", "f4-v1.0.0", "f4-v1.1.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Installing different tool with name clash should fail",
						do: func(t *testing.T) {
							testutil.NotOk(t, g.ExpectErr(p.root, goBinPath, "get", "golang.org/x/totally-not-tools/cmd/goimports@cb1345f3a375367f8439bba882e90348348288d9"))
						},
						expectBinaries: []string{"f2-clone-v1.3.0", "f2-clone-v1.4.0", "f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "f3-v1.3.0", "f3-v1.4.0", "f4-v1.0.0", "f4-v1.1.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},

					{
						name: "Updating f4 to multiple versions with none should fail",
						do: func(t *testing.T) {
							testutil.NotOk(t, g.ExpectErr(p.root, goBinPath, "get", "f2@v1.4.0,v1.1.0,none"))
						},
						expectBinaries: []string{"f2-clone-v1.3.0", "f2-clone-v1.4.0", "f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "f3-v1.3.0", "f3-v1.4.0", "f4-v1.0.0", "f4-v1.1.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Updating f4 back to non array version should work",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "f4@v1.1.0"))
						},
						expectBinaries: []string{"f2-clone-v1.3.0", "f2-clone-v1.4.0", "f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "f3-v1.3.0", "f3-v1.4.0", "f4-v1.0.0", "f4-v1.1.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Remove goimports2 by name",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "goimports2@none"))
							testutil.Equals(t, "Name\t\t\tBinary Name\t\t\t\t\t\t\tPackage @ Version\t\t\t\t\t\t\t\t\t\t\n----\t\t\t-----------\t\t\t\t\t\t\t-----------------\t\t\t\t\t\t\t\t\t\t\nf4\t\t\tf4-v1.1.0\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.1.0\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.1.0\t\t\t\t\t\tgithub.com/fatih/faillint@v1.1.0\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.0.0\t\t\t\t\t\tgithub.com/fatih/faillint@v1.0.0\t\t\t\t\t\t\t\ngo-bindata\tgo-bindata-v3.1.1+incompatible\t\t\tgithub.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible\t\ngoimports\t\tgoimports-v0.0.0-20200522201501-cb1345f3a375\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375", g.ExecOutput(t, p.root, goBinPath, "list"))
						},
						expectBinaries: []string{"f2-clone-v1.3.0", "f2-clone-v1.4.0", "f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "f3-v1.3.0", "f3-v1.4.0", "f4-v1.0.0", "f4-v1.1.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Remove goimports by path",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "golang.org/x/tools/cmd/goimports@none"))
						},
						expectBinaries: []string{"f2-clone-v1.3.0", "f2-clone-v1.4.0", "f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "f3-v1.3.0", "f3-v1.4.0", "f4-v1.0.0", "f4-v1.1.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Remove faillint by name",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "faillint@none"))
						},
						expectBinaries: []string{"f2-clone-v1.3.0", "f2-clone-v1.4.0", "f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "f3-v1.3.0", "f3-v1.4.0", "f4-v1.0.0", "f4-v1.1.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Remove f4 by name",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "f4@none"))
						},
						expectBinaries: []string{"f2-clone-v1.3.0", "f2-clone-v1.4.0", "f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "f3-v1.3.0", "f3-v1.4.0", "f4-v1.0.0", "f4-v1.1.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
					{
						name: "Remove go-bindata by name",
						do: func(t *testing.T) {
							fmt.Println(g.ExecOutput(t, p.root, goBinPath, "get", "go-bindata@none"))
							testutil.Equals(t, "Name\tBinary Name\tPackage @ Version\t\n----\t-----------\t-----------------", g.ExecOutput(t, p.root, goBinPath, "list"))
						},
						expectBinaries: []string{"f2-clone-v1.3.0", "f2-clone-v1.4.0", "f2-v1.0.0", "f2-v1.1.0", "f2-v1.2.0", "f2-v1.3.0", "f2-v1.4.0", "f2-v1.5.0", "f3-v1.3.0", "f3-v1.4.0", "f4-v1.0.0", "f4-v1.1.0", "faillint-v1.0.0", "faillint-v1.1.0", "faillint-v1.3.0", "faillint-v1.4.0", "faillint-v1.5.0", "go-bindata-v3.1.1+incompatible", "goimports-v0.0.0-20200521211927-2b542361a4fc", "goimports-v0.0.0-20200522201501-cb1345f3a375", "goimports2-v0.0.0-20200515010526-7d3b6ebf133d", "goimports2-v0.0.0-20200519175826-7521f6f42533"},
					},
				} {
					if ok := t.Run(tcase.name, func(t *testing.T) {
						defer p.assertNotChanged(t, defaultModDir)

						tcase.do(t)

						expectBingoListRows(t, tcase.expectRows, g.ExecOutput(t, p.root, goBinPath, "list"))
						testutil.Equals(t, tcase.expectBinaries, g.existingBinaries(t))
					}); !ok {
						return
					}
				}
			}); !ok {
				return
			}
		}
	}); !ok {
		return
	}
	t.Run("Compatibility test", func(t *testing.T) {
		t.Skip("f")
		dirs, err := filepath.Glob("testdata/testproject*")
		testutil.Ok(t, err)
		for _, dir := range dirs {
			t.Run(dir, func(t *testing.T) {
				for _, isGoProject := range []bool{false, true} {
					t.Run(fmt.Sprintf("isGoProject=%v", isGoProject), func(t *testing.T) {
						t.Run("Via bingo get all", func(t *testing.T) {
							g.Clear(t)

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
							g.Clear(t)

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
							g.Clear(t)

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

							g.Clear(t)

							// We manually build bingo binary to make sure GOCACHE will not hit us.
							goBinPath := filepath.Join(g.tmpDir, bingoBin)
							buildInitialGobin(t, goBinPath)

							// Copy testproject at the beginning to temp dir.
							prjRoot := filepath.Join(g.tmpDir, "testproject")
							p := newTestProject(t, dir, prjRoot, isGoProject)
							p.assertNotChanged(t, defaultModDir)

							testutil.Equals(t, []string{}, g.existingBinaries(t))
							g.ExecOutput(t, p.root, makePath, "faillint-exists")
							g.ExecOutput(t, p.root, makePath, "goimports-exists")
							g.ExecOutput(t, p.root, makePath, "goimports2-exists")

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

type row struct {
	name, binName, pkgVersion string
}

func expectBingoListRows(t testing.TB, expect []row, output string) {
	t.Helper()

	trimmed := strings.TrimLeft(output, "Name\tBinary Name\tPackage @ Version\n----\t-----------\t-----------------\n")
	var got []row
	for _, line := range strings.Split(trimmed, "\n") {
		s := strings.Fields(line)
		r := row{name: s[0]}
		if len(s) > 1 {
			r.binName = s[1]
		}
		if len(s) > 2 {
			r.pkgVersion = s[2]
		}
		got = append(got, r)
	}
	testutil.Equals(t, expect, got)
}

func TestExpectBingoListRows(t *testing.T) {
	expectBingoListRows(t, []row{
		{name: "f4", binName: "f4-v1.1.0", pkgVersion: "github.com/fatih/faillint@v1.1.0"},
		{name: "faillint", binName: "faillint-v1.1.0", pkgVersion: "github.com/fatih/faillint@v1.1.0"},
		{name: "faillint", binName: "faillint-v1.0.0", pkgVersion: "github.com/fatih/faillint@v1.0.0"},
		{name: "go-bindata", binName: "go-bindata-v3.1.1+incompatible", pkgVersion: "github.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible"},
		{name: "goimports", binName: "goimports-v0.0.0-20200522201501-cb1345f3a375", pkgVersion: "golang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375"},
	}, "Name\t\t\tBinary Name\t\t\t\t\t\t\tPackage @ Version\n----\t\t\t-----------\t\t\t\t\t\t\t-----------------\nf4\t\t\tf4-v1.1.0\t\t\t\t\t\t\t\tgithub.com/fatih/faillint@v1.1.0\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.1.0\t\t\t\t\t\tgithub.com/fatih/faillint@v1.1.0\t\t\t\t\t\t\t\nfaillint\t\tfaillint-v1.0.0\t\t\t\t\t\tgithub.com/fatih/faillint@v1.0.0\t\t\t\t\t\t\t\ngo-bindata\tgo-bindata-v3.1.1+incompatible\t\t\tgithub.com/go-bindata/go-bindata/go-bindata@v3.1.1+incompatible\t\ngoimports\t\tgoimports-v0.0.0-20200522201501-cb1345f3a375\tgolang.org/x/tools/cmd/goimports@v0.0.0-20200522201501-cb1345f3a375")
}
