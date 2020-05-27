// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/bwplotka/bingo/pkg/bingo"
	"github.com/bwplotka/bingo/pkg/gomodcmd"
	"github.com/pkg/errors"
)

type getConfig struct {
	runner *gomodcmd.Runner
	modDir string
	update gomodcmd.GetUpdatePolicy
	name   string

	// target name or target package path, optionally with version(s).
	rawTarget string
}

func get(
	ctx context.Context,
	c getConfig,
) (err error) {
	if c.rawTarget == "" {
		// Empty means all.
		if c.name != "" {
			return errors.New("name cannot by specified if no target was given")
		}
		modules, err := filepath.Glob(filepath.Join(c.modDir, "*.mod"))
		if err != nil {
			return err
		}
		for _, m := range modules {
			mc := c
			mc.rawTarget, _ = bingo.NameFromModFile(m)
			if err := get(ctx, mc); err != nil {
				return err
			}
		}
		return nil
	}

	var modVersions []string
	s := strings.Split(c.rawTarget, "@")
	nameOrPackage := s[0]
	if len(s) > 1 {
		modVersions = strings.Split(s[1], ",")
	}

	if len(modVersions) > 1 {
		for _, v := range modVersions {
			if v == "none" {
				return errors.Errorf("none is not allowed when there are more than one specified version, got: %v", modVersions)
			}
		}
	}

	if len(modVersions) == 0 {
		modVersions = append(modVersions, "")
	}

	pkgPath := nameOrPackage
	name := nameOrPackage
	if !strings.Contains(nameOrPackage, "/") {
		// Binary referenced by name, get full package name if module file exists.
		pkgPath, err = binNameToPackagePath(nameOrPackage, c.modDir)
		if err != nil {
			return err
		}

		if c.name != "" && c.name != name {
			// Rename requested. Remove old mod(s) in this case, but only at the end.
			defer func() { _ = removeAllGlob(filepath.Join(c.modDir, name+".*")) }()
		}
	} else {
		// Binary referenced by path, get default name from package path.
		name = path.Base(pkgPath)
	}

	if c.name != "" {
		name = c.name
	}

	binModFiles, err := filepath.Glob(filepath.Join(c.modDir, name+".*.mod"))
	if err != nil {
		return err
	}
	binModFiles = append([]string{filepath.Join(c.modDir, name+".mod")}, binModFiles...)

	if modVersions[0] == "none" {
		// none means we no longer want to version this package.
		// NOTE: We don't remove binaries.
		return removeAllGlob(filepath.Join(c.modDir, name+".*"))
	}

	for i, v := range modVersions {
		if err := getOne(ctx, c, i, v, pkgPath, name); err != nil {
			return errors.Wrapf(err, "%d: getting %s", i, v)
		}
	}

	// Remove unused mod files.
	for i := len(binModFiles); i > 0 && i > len(modVersions); i-- {
		if err := os.RemoveAll(filepath.Join(c.modDir, fmt.Sprintf("%s.%d.mod", name, i-1))); err != nil {
			return err
		}
	}
	return nil
}

func getOne(
	ctx context.Context,
	c getConfig,
	i int,
	version string,
	pkgPath string,
	name string,
) (err error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	// The out module file we generate/maintain keep in modDir.
	outModFile := filepath.Join(c.modDir, name+".mod")
	if i > 0 {
		outModFile = filepath.Join(c.modDir, fmt.Sprintf("%s.%d.mod", name, i))
	}

	// Check if module exists and has bingo watermark, otherwise assume it's malformed and remove.
	outExists, err := bingo.ModHasMeta(outModFile, nil)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if !outExists {
		if err := os.RemoveAll(outModFile); err != nil {
			return err
		}
	}

	runnable := c.runner.With(ctx, outModFile, c.modDir)
	if err := ensureModFileExists(runnable, outModFile); err != nil {
		return err
	}

	if !outExists || version != "" || c.update != gomodcmd.NoUpdatePolicy {
		// Steps 1 & 2: Resolve and download (if needed) thanks to 'go get' on our separate .mod file.
		targetWithVer := pkgPath
		if version != "" {
			targetWithVer = fmt.Sprintf("%s@%s", pkgPath, version)
		}
		if err := runnable.GetD(c.update, targetWithVer); err != nil {
			return err
		}
	}
	if !outExists {
		// Add our metadata to pkgPath module file only if it did not exists before go get.
		if err := bingo.AddMetaToMod(outModFile, pkgPath); err != nil {
			return errors.Wrap(err, "adding meta")
		}
	}

	// Check if path is pointing to non-buildable package. Fail it is non-buildable. Hacky!
	if listOutput, err := runnable.List("-f={{.Name}}", pkgPath); err != nil {
		return err
	} else if !strings.HasSuffix(listOutput, "main") {
		return errors.Errorf("package %s is non-main (go list output %q), nothing to get and build", pkgPath, listOutput)
	}

	// Refetch version to ensure we have correct one.
	_, version, err = bingo.ModDirectPackage(outModFile, nil)
	if err != nil {
		return err
	}
	// Step 3: Build and install.
	return runnable.Build(pkgPath, fmt.Sprintf("%s-%s", name, version))
}

func binNameToPackagePath(binary string, modDir string) (string, error) {
	currModFile := filepath.Join(modDir, binary+".mod")

	// Get full import path from module file which has module and encoded sub path.
	if _, err := os.Stat(currModFile); err != nil {
		if os.IsNotExist(err) {
			return "", errors.Errorf("binary %q was not installed before. Use full package name to install it", binary)
		}
		return "", err
	}

	m, _, err := bingo.ModDirectPackage(currModFile, nil)
	if err != nil {
		return "", errors.Wrapf(err, "binary %q was installed, but go modules %s is malformed. Use full package name to reinstall it", binary, currModFile)
	}
	return m, nil
}

const modREADMEFmt = `# Project Development Dependencies.

This is directory which stores Go modules with pinned buildable package that is used within this repository, managed by https://github.com/bwplotka/bingo.

* Run ` + "`" + "bingo get" + "`" + ` to install all tools having each own module file in this directory.
* Run ` + "`" + "bingo get <tool>" + "`" + ` to install <tool> that have own module file in this directory.
* If ` + "`" + bingo.MakefileBinVarsName + "`" + ` is present, use $(<upper case tool name>) variable where <tool> is the %s/<tool>.mod.
* See https://github.com/bwplotka/bingo or -h on how to add, remove or change binaries dependencies.

## Requirements

* Go 1.14+
`

const gitignore = `
# Ignore everything
*

# But not these files:
!.gitignore
!*.mod
!*.md
!*.mk
`

func ensureModFileExists(r gomodcmd.Runnable, modFile string) error {
	if err := os.MkdirAll(filepath.Dir(modFile), os.ModePerm); err != nil {
		return errors.Wrapf(err, "create moddir %s", filepath.Dir(modFile))
	}

	// README.
	if err := ioutil.WriteFile(
		filepath.Join(filepath.Dir(modFile), "README.md"),
		[]byte(fmt.Sprintf(modREADMEFmt, filepath.Join("<root>", filepath.Base(filepath.Dir(modFile))))),
		os.ModePerm,
	); err != nil {
		return err
	}
	// gitignore.
	if err := ioutil.WriteFile(
		filepath.Join(filepath.Dir(modFile), ".gitignore"),
		[]byte(gitignore),
		os.ModePerm,
	); err != nil {
		return err
	}

	_, err := os.Stat(modFile)
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "stat module file %s", modFile)
	}
	if err == nil {
		return nil
	}
	// Module name does not matter.
	return errors.Wrap(r.ModInit("_"), "mod init")
}

func removeAllGlob(glob string) error {
	files, err := filepath.Glob(glob)
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := os.RemoveAll(f); err != nil {
			return err
		}
	}
	return nil
}
