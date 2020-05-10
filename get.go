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

	"github.com/bwplotka/gobin/pkg/gobin"
	"github.com/bwplotka/gobin/pkg/gomodcmd"
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
			return "", errors.Errorf("Binary %s was not before installed. Use full package name to install it", binary)
		}
		return "", err
	}

	m, err := gobin.ModDirectPackage(currModFile, nil)
	if err != nil {
		return "", errors.Wrapf(err, "Binary %s was installed, but go modules %s is malformed. Use full package name to reinstall it.", binary, currModFile)
	}
	if m == "" {
		return "", errors.Errorf("Binary %s was not before installed. Use full package name to install it", binary)
	}
	return m, nil
}

const modREADMEFmt = `# Development Dependencies.

This is directory which stores Go modules for each tools that is used within this repository, managed by https://github.com/bwplotka/gobin.

## Requirements

* Network (:
* Go 1.14+

## Usage

Just run ` + "`" + "go get -modfile %s/<tool>.mod" + "`" + `to install tool in required version in your $(GOBIN).

### Within Makefile

Use $(<tool>) variable where <tool> is the %s/<tool>.mod.

This directory is managed by gobin tool.

* Run ` + "`" + "go get -modfile %s/gobin.mod" + "`" + ` if you did not before to install gobin.
* Run ` + "`" + "gobin get" + "`" + ` to install all tools in this directory.
* See https://github.com/bwplotka/gobin or -h on how to add, remove or change binaries dependencies.
`

func ensureModFileExists(r gomodcmd.Runnable, modFile string) error {
	_, err := os.Stat(modFile)
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "stat module file %s", modFile)
	}
	if err == nil {
		return nil
	}

	// ModuleFile does not exists, ensure directory and README.md exists.
	if err := os.MkdirAll(filepath.Dir(modFile), os.ModePerm); err != nil {
		return errors.Wrapf(err, "create moddir %s", filepath.Dir(modFile))
	}

	// Be nice to people.
	readmePath := filepath.Join(filepath.Dir(modFile), "README.md")
	if _, err := os.Stat(readmePath); err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrapf(err, "stat readme %s", modFile)
		}

		relDir := filepath.Join("<root>", filepath.Base(filepath.Dir(modFile)))
		if err := ioutil.WriteFile(readmePath, []byte(fmt.Sprintf(modREADMEFmt, relDir, relDir, relDir)), os.ModePerm); err != nil {
			return err
		}
	}
	// Module name does not matter.
	return errors.Wrap(r.ModInit("_"), "mod init")
}

func getOne(
	ctx context.Context,
	r *gomodcmd.Runner,
	modDir string,
	update gomodcmd.GetUpdatePolicy,
	binOrPackage string,
	output string,
) (err error) {
	modVer := ""
	s := strings.Split(binOrPackage, "@")
	if len(s) > 1 {
		modVer = s[1]
	}
	binOrPackage = s[0]

	pkgPath := binOrPackage
	if !strings.Contains(binOrPackage, "/") {
		pkgPath, err = binNameToPackagePath(binOrPackage, modDir)
		if err != nil {
			return err
		}
		// At this point we know <binary>.mod exists.

		if modVer == "none" {
			// none means we no longer want to version this package.
			return os.RemoveAll(filepath.Join(modDir, binOrPackage+".mod"))
		}

		// Record binary name used as long as rename was not expected. This is what it was configured with before.
		// TODO(bwplotka): Test this logic, there are many cases.
		if output != "" {
			output = binOrPackage
		}
	} else {
		if modVer == "none" {
			// none means we no longer want to version this package.
			return os.RemoveAll(filepath.Join(modDir, path.Base(pkgPath)+".mod"))
		}
	}

	if output == "" {
		output = path.Base(pkgPath)
	}

	if output != binOrPackage {
		// There might be case of rename. Remove old mod in this case, but only at once all is successful here.
		defer func() { _ = os.RemoveAll(filepath.Join(modDir, binOrPackage+".mod")) }()
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	tmpDir, err := ioutil.TempDir(os.TempDir(), "gobin")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// The out module file we generate/maintain keep in modDir.
	outModFile := filepath.Join(modDir, output+".mod")

	// Check if module exists and has gobin watermark, otherwise assume it's malformed and remove.
	outExists, err := gobin.ModHasMeta(outModFile, nil)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if !outExists {
		if err := os.RemoveAll(outModFile); err != nil {
			return err
		}
	}

	runnable := r.With(ctx, outModFile, tmpDir)

	if err := ensureModFileExists(runnable, outModFile); err != nil {
		return err
	}

	// Step 0: Copy existing module file to go.mod and create fake .go code file that imports package for Go Modules to work smoothly.
	if err := ioutil.WriteFile(filepath.Join(tmpDir, "go.mod"), nil, os.ModePerm); err != nil {
		return err
	}
	if err := gobin.CreateGoFileWithPackages(filepath.Join(tmpDir, output+".go"), pkgPath); err != nil {
		return err
	}

	{
		// Steps 1 & 2: Resolve and download (if needed) thanks to 'go get' on ou separate .mod file.
		targetWithVer := pkgPath
		if modVer != "" {
			targetWithVer = fmt.Sprintf("%s@%s", pkgPath, modVer)
		}
		if err := runnable.GetD(update, targetWithVer); err != nil {
			return err
		}
	}

	// Check if path is pointing to non-buildable package, then fail.
	pkgName, err := runnable.List("-f={{.Name}}", pkgPath)
	if err != nil {
		return err
	}
	if pkgName != "main" {
		return errors.Errorf("package %s is non-main (found %q), nothing to get and build", pkgPath, pkgName)
	}

	// Step 3: Build and install.
	if err := runnable.Build(pkgPath, output); err != nil {
		return err
	}
	{
		// Step 4: tidy.
		if err := runnable.ModTidy(); err != nil {
			return errors.Wrap(err, "mod tidy")
		}

		if !outExists {
			// Add our metadata to pkgPath module file only if it did not exists before go get.
			if err := gobin.AddMetaToMod(outModFile, pkgPath); err != nil {
				return errors.Wrap(err, "adding meta")
			}
		}
	}
	return nil
}

func ensureGobinModFile(
	ctx context.Context,
	r *gomodcmd.Runner,
	modDir string,
) error {
	_, err := os.Stat(filepath.Join(modDir, gobinBinName+".mod"))
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "stat module file %s", filepath.Join(modDir, gobinBinName))
	}
	if err == nil {
		return nil
	}

	// Pin the latest version.
	// TODO(bwplotka): Considering pinning the version that user use now?
	return getOne(ctx, r, modDir, gomodcmd.UpdatePolicy, gobinInstallPath, gobinBinName)
}
