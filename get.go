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

func get(
	ctx context.Context,
	r *gomodcmd.Runner,
	modDir string,
	update gomodcmd.GetUpdatePolicy,
	target string,
	output string,
) (err error) {
	if target == "" {
		// Empty means all.
		modules, err := filepath.Glob(filepath.Join(modDir, "*.mod"))
		if err != nil {
			return err
		}

		for _, m := range modules {
			m = strings.TrimSuffix(filepath.Base(m), ".mod")
			if err := getOne(ctx, r, modDir, update, m, output); err != nil {
				return err
			}
		}
		return nil
	}
	return getOne(ctx, r, modDir, update, target, output)
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
* If ` + "`" + "Makefile.binary-variables" + "`" + ` is present, use $(<upper case tool name>) variable where <tool> is the %s/<tool>.mod.
* See https://github.com/bwplotka/bingo or -h on how to add, remove or change binaries dependencies.

## Requirements

* Go 1.14+
`

func ensureModFileExists(r gomodcmd.Runnable, modFile string) error {
	if err := os.MkdirAll(filepath.Dir(modFile), os.ModePerm); err != nil {
		return errors.Wrapf(err, "create moddir %s", filepath.Dir(modFile))
	}

	if err := ioutil.WriteFile(
		filepath.Join(filepath.Dir(modFile), "README.md"),
		[]byte(fmt.Sprintf(modREADMEFmt, filepath.Join("<root>", filepath.Base(filepath.Dir(modFile))))),
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

func getOne(
	ctx context.Context,
	r *gomodcmd.Runner,
	modDir string,
	update gomodcmd.GetUpdatePolicy,
	nameOrPackage string,
	output string,
) (err error) {
	modVer := ""
	s := strings.Split(nameOrPackage, "@")
	if len(s) > 1 {
		modVer = s[1]
	}

	nameOrPackage = s[0]
	if modVer == "none" {
		binary := path.Base(nameOrPackage)
		if _, err := os.Stat(filepath.Join(modDir, binary+".mod")); err != nil {
			if os.IsNotExist(err) {
				return errors.Errorf("binary %q was not installed before, nothing to remove", binary)
			}
			return err
		}
		// none means we no longer want to version this package.
		// NOTE: We don't remove binaries.
		return removeAllGlob(filepath.Join(modDir, binary+".*"))
	}

	pkgPath := nameOrPackage
	if !strings.Contains(nameOrPackage, "/") {
		// Binary referenced by name, get full package name if module file exists.
		pkgPath, err = binNameToPackagePath(nameOrPackage, modDir)
		if err != nil {
			return err
		}
		if output == "" {
			output = nameOrPackage
		} else if output != nameOrPackage {
			// There might be case of rename. Remove old mod in this case, but only at the end.
			defer func() { _ = removeAllGlob(filepath.Join(modDir, nameOrPackage+".*")) }()
		}
	} else if output == "" {
		output = path.Base(pkgPath)
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	// The out module file we generate/maintain keep in modDir.
	outModFile := filepath.Join(modDir, output+".mod")

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

	runnable := r.With(ctx, outModFile, modDir)
	if err := ensureModFileExists(runnable, outModFile); err != nil {
		return err
	}

	if modVer != "" || update != gomodcmd.NoUpdatePolicy {
		// Steps 1 & 2: Resolve and download (if needed) thanks to 'go get' on our separate .mod file.
		targetWithVer := pkgPath
		if modVer != "" {
			targetWithVer = fmt.Sprintf("%s@%s", pkgPath, modVer)
		}
		if err := runnable.GetD(update, targetWithVer); err != nil {
			return err
		}
	}

	// Check if path is pointing to non-buildable package. Fail it is non-buildable.
	listOutput, err := runnable.List("-f={{.Name}}", pkgPath)
	if err != nil {
		return err
	}
	// Hacky.
	if !strings.HasSuffix(listOutput, "main") {
		return errors.Errorf("package %s is non-main (go list output %q), nothing to get and build", pkgPath, listOutput)
	}

	// Step 3: Build and install.
	if err := runnable.Build(pkgPath, output); err != nil {
		return err
	}
	{
		if !outExists {
			// Add our metadata to pkgPath module file only if it did not exists before go get.
			if err := bingo.AddMetaToMod(outModFile, pkgPath); err != nil {
				return errors.Wrap(err, "adding meta")
			}
		}
	}
	return nil
}
