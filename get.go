// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
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
	runner    *gomodcmd.Runner
	modDir    string
	relModDir string
	update    gomodcmd.GetUpdatePolicy
	name      string

	// target name or target package path, optionally with Version(s).
	rawTarget string
}

func get(
	ctx context.Context,
	logger *log.Logger,
	c getConfig,
) (err error) {
	if c.rawTarget == "" {
		// Empty means all.
		if c.name != "" {
			return errors.New("name cannot by specified if no target was given")
		}
		modFiles, err := bingoModFiles(c.modDir)
		if err != nil {
			return err
		}
		for _, m := range modFiles {
			mc := c
			mc.rawTarget, _ = bingo.NameFromModFile(m)
			if err := get(ctx, logger, mc); err != nil {
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
				return errors.Errorf("none is not allowed when there are more than one specified Version, got: %v", modVersions)
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
		pkgPath, err = packagePathFromBinaryName(nameOrPackage, c.modDir)
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

	if name == strings.TrimSuffix(fakeRootModFileName, ".mod") {
		return errors.New("requested binary with name `go`. This is impossible, choose different name using -name flag.")
	}

	binModFiles, err := filepath.Glob(filepath.Join(c.modDir, name+".*.mod"))
	if err != nil {
		return err
	}
	binModFiles = append([]string{filepath.Join(c.modDir, name+".mod")}, binModFiles...)

	if modVersions[0] == "none" {
		// none means we no longer want to Version this package.
		// NOTE: We don't remove binaries.
		return removeAllGlob(filepath.Join(c.modDir, name+".*"))
	}

	for i, v := range modVersions {
		if err := getOne(ctx, logger, c, i, v, pkgPath, name); err != nil {
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

func cleanGoGetTmpFiles(modDir string) error {
	// Remove all sum and tmp files
	if err := removeAllGlob(filepath.Join(modDir, "*.sum")); err != nil {
		return err
	}
	return removeAllGlob(filepath.Join(modDir, "*.tmp.*"))
}

func getOne(
	ctx context.Context,
	logger *log.Logger,
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
	// Cleanup all for fresh start.
	if err := cleanGoGetTmpFiles(c.modDir); err != nil {
		return err
	}
	if err := ensureModDirExists(logger, c.relModDir); err != nil {
		return errors.Wrap(err, "ensure mod dir")
	}
	// Set up tmp file that we will work on for now.
	// This is to avoid partial updates.
	tmpModFile := filepath.Join(c.modDir, name+".tmp.mod")
	emptyModFile, err := createTmpModFileFromExisting(ctx, c.runner, outModFile, tmpModFile)
	if err != nil {
		return errors.Wrap(err, "create tmp mod file")
	}

	runnable := c.runner.With(ctx, tmpModFile, c.modDir)
	if version != "" || emptyModFile || c.update != gomodcmd.NoUpdatePolicy {
		// Steps 1 & 2: Resolve and download (if needed) thanks to 'go get' on our separate .mod file.
		targetWithVer := pkgPath
		if version != "" {
			targetWithVer = fmt.Sprintf("%s@%s", pkgPath, version)
		}

		if err := runnable.GetD(c.update, targetWithVer); err != nil {
			return errors.Wrap(err, "go get -d")
		}
	}
	if err := bingo.EnsureModMeta(tmpModFile, pkgPath); err != nil {
		return errors.Wrap(err, "ensuring meta")
	}

	// Check if path is pointing to non-buildable package. Fail it is non-buildable. Hacky!
	if listOutput, err := runnable.List("-f={{.Name}}", pkgPath); err != nil {
		return err
	} else if !strings.HasSuffix(listOutput, "main") {
		return errors.Errorf("package %s is non-main (go list output %q), nothing to get and build", pkgPath, listOutput)
	}

	// Refetch Version to ensure we have correct one.
	_, version, err = bingo.ModDirectPackage(tmpModFile, nil)
	if err != nil {
		return errors.Wrap(err, "get direct package")
	}

	// We were working on tmp file, do atomic rename.
	if err := os.Rename(tmpModFile, outModFile); err != nil {
		return errors.Wrap(err, "rename")
	}
	// Step 3: Build and install.
	return c.runner.With(ctx, outModFile, c.modDir).Build(pkgPath, fmt.Sprintf("%s-%s", name, version))
}

func packagePathFromBinaryName(binary string, modDir string) (string, error) {
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
* For Makefile: Make sure to put ` + "`" + "include %s/" + bingo.MakefileBinVarsName + "`" + ` in your Makefile, then use $(<upper case tool name>) variable where <tool> is the %s/<tool>.mod.
* For shell: Run ` + "`" + "source %s/" + bingo.EnvBinVarsName + "`" + ` to source all environment variable for each tool
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
!README.md
!Variables.mk
!variables.env

*tmp.mod
`

func ensureModDirExists(logger *log.Logger, relModDir string) error {
	_, err := os.Stat(relModDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrapf(err, "stat bingo module dir %s", relModDir)
		}

		logger.Printf("Bingo not used before here, creating directory for pinned modules for you at %s\n", relModDir)
		if err := os.MkdirAll(relModDir, os.ModePerm); err != nil {
			return errors.Wrapf(err, "create moddir %s", relModDir)
		}
	}

	// Hack against:
	// "A file named go.mod must still be present in order to determine the module root directory, but it is not accessed."
	// Ref: https://golang.org/doc/go1.14#go-flags
	// TODO(bwplotka): Remove it: https://github.com/bwplotka/bingo/issues/20
	if err := ioutil.WriteFile(
		filepath.Join(relModDir, fakeRootModFileName),
		[]byte("module _ // Fake go.mod auto-created by 'bingo' for go -moddir compatibility with non-Go projects. Commit this file, together with other .mod files."),
		os.ModePerm,
	); err != nil {
		return err
	}

	// README.
	if err := ioutil.WriteFile(
		filepath.Join(relModDir, "README.md"),
		[]byte(fmt.Sprintf(modREADMEFmt, relModDir, relModDir, relModDir)),
		os.ModePerm,
	); err != nil {
		return err
	}
	// gitignore.
	return ioutil.WriteFile(
		filepath.Join(relModDir, ".gitignore"),
		[]byte(gitignore),
		os.ModePerm,
	)
}

func createTmpModFileFromExisting(ctx context.Context, r *gomodcmd.Runner, modFile, tmpModFile string) (emptyModFile bool, _ error) {
	if err := os.RemoveAll(tmpModFile); err != nil {
		return false, errors.Wrap(err, "rm")
	}

	_, err := os.Stat(modFile)
	if err != nil && !os.IsNotExist(err) {
		return false, errors.Wrapf(err, "stat module file %s", modFile)
	}
	if err == nil {
		return false, copyFile(modFile, tmpModFile)
	}
	return true, errors.Wrap(r.With(ctx, tmpModFile, filepath.Dir(modFile)).ModInit("_"), "mod init")
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	// TODO(bwplotka): Check those errors in defer.
	defer source.Close()
	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	buf := make([]byte, 1024)
	for {
		n, err := source.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		if _, err := destination.Write(buf[:n]); err != nil {
			return err
		}
	}
	return nil
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
