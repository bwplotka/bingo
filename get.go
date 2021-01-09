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
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwplotka/bingo/pkg/bingo"
	"github.com/bwplotka/bingo/pkg/runner"
	"github.com/efficientgo/tools/core/pkg/errcapture"
	"github.com/efficientgo/tools/pkg/merrors"
	"github.com/pkg/errors"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

var goModVersionRegexp = regexp.MustCompile("^v[0-9]*$")

type getConfig struct {
	runner    *runner.Runner
	modDir    string
	relModDir string
	update    runner.GetUpdatePolicy
	name      string
	rename    string

	// target name or target package path, optionally with Version(s).
	rawTarget string
}

func parseTarget(rawTarget string) (name string, pkgPath string, versions []string, err error) {
	if rawTarget == "" {
		return "", "", nil, errors.New("target is empty, this should be filtered earlier")
	}

	s := strings.Split(rawTarget, "@")
	nameOrPackage := s[0]
	if len(s) > 1 {
		versions = strings.Split(s[1], ",")
	}

	if len(versions) > 1 {
		// Check for duplicates or/and none.
		dup := map[string]struct{}{}
		for _, v := range versions {
			if _, ok := dup[v]; ok {
				return "", "", nil, errors.Errorf("version duplicates are not allowed, got: %v", versions)
			}
			dup[v] = struct{}{}
			if v == "none" {
				return "", "", nil, errors.Errorf("none is not allowed when there are more than one specified Version, got: %v", versions)
			}
		}
	}

	name = nameOrPackage
	if strings.Contains(nameOrPackage, "/") {
		// Binary referenced by path, get default name from package path.
		pkgPath = nameOrPackage
		name = path.Base(pkgPath)
		if pkgSplit := strings.Split(pkgPath, "/"); len(pkgSplit) > 3 && goModVersionRegexp.MatchString(name) {
			// It's common pattern to name urls with versions in go modules. Exclude that.
			name = pkgSplit[len(pkgSplit)-2]
		}
	}
	return name, pkgPath, versions, nil
}

func getAll(
	ctx context.Context,
	logger *log.Logger,
	c getConfig,
) (err error) {
	if c.name != "" {
		return errors.New("name cannot by specified if no target was given")
	}
	if c.rename != "" {
		return errors.New("rename cannot by specified if no target was given")
	}

	pkgs, err := bingo.ListPinnedMainPackages(logger, c.relModDir, false)
	if err != nil {
		return err
	}
	for _, p := range pkgs {
		mc := c
		mc.rawTarget = p.Name
		if len(p.Versions) > 1 {
			// Compose array target. Order of versions matter.
			var versions []string
			for _, v := range p.Versions {
				versions = append(versions, v.Version)
			}
			mc.rawTarget += "@" + strings.Join(versions, ",")
		}
		if err := get(ctx, logger, mc); err != nil {
			return err
		}
	}
	return nil
}

func get(ctx context.Context, logger *log.Logger, c getConfig) (err error) {
	if c.rawTarget == "" {
		// Empty means get all. It recursively invokes get for each existing binary.
		return getAll(ctx, logger, c)
	}

	name, pkgPath, versions, err := parseTarget(c.rawTarget)
	if err != nil {
		return errors.Wrapf(err, "parse %v", c.rawTarget)
	}

	targetName := name
	if c.name != "" {
		targetName = c.name
	}

	existingModFiles, err := filepath.Glob(filepath.Join(c.modDir, targetName+".mod"))
	if err != nil {
		return err
	}
	existingModArrFiles, err := filepath.Glob(filepath.Join(c.modDir, targetName+".*.mod"))
	if err != nil {
		return err
	}
	existingModFiles = append(existingModFiles, existingModArrFiles...)

	// Get full import path from any existing module file for this name.
	if pkgPath == "" && len(existingModFiles) == 0 {
		// Binary referenced by name, get full package name if module file exists.
		return errors.Errorf("binary %q was not installed before. Use full package name to install it", targetName)
	}

	if len(existingModFiles) > 0 {
		modPkgPath, _, err := bingo.ModDirectPackage(existingModFiles[0])
		if err != nil {
			return errors.Wrapf(err, "binary %q was installed, but go modules %s is malformed. Use full package name to reinstall it", targetName, existingModFiles[0])
		}

		if pkgPath != "" && pkgPath != modPkgPath {
			// Check if someone is not referencing totally different path.
			return errors.Errorf("failed to install %q under %q name as binary with the same name is already installed for path %q. "+
				"Uninstall exsiting %q tool using `@none` or use `-n` flag to choose different name", pkgPath, targetName, modPkgPath, targetName)
		}
		pkgPath = modPkgPath
	}

	if c.rename != "" {
		targetName = c.rename
	}

	if targetName == "cmd" {
		return errors.Errorf("package %s would be installed with ambiguous name %s. This is a common, but slightly annoying package layout"+
			"It's advised to choose unique name with -n flag", pkgPath, targetName)
	}
	if targetName == strings.TrimSuffix(bingo.FakeRootModFileName, ".mod") {
		return errors.Errorf("requested binary with name %q`. This is impossible, choose different name using -n flag", strings.TrimSuffix(bingo.FakeRootModFileName, ".mod"))
	}

	if len(versions) == 0 {
		for _, f := range existingModFiles {
			_, version, err := bingo.ModDirectPackage(f)
			if err != nil {
				return errors.Wrapf(err, "binary %q was installed, but go modules %s is malformed Use full package name to reinstall it", name, existingModFiles[0])
			}
			versions = append(versions, version)
		}

		if len(versions) == 0 {
			// Binary never seen before and requested to be installed from latest.
			versions = append(versions, "")
		}
	} else if versions[0] == "none" {
		// none means we no longer want to Version this package.
		// NOTE: We don't remove binaries.
		return removeAllGlob(filepath.Join(c.modDir, name+".*"))
	}

	for i, version := range versions {
		if err := getOne(ctx, logger, c, i, pkgPath, targetName, version); err != nil {
			return errors.Wrapf(err, "%d: getting %s", i, version)
		}
	}

	if c.rename != "" && targetName != name {
		// Rename requested. Remove old mod(s) in this case, but only at the end.
		defer func() { _ = removeAllGlob(filepath.Join(c.modDir, name+".*")) }()
	}

	// Remove target unused arr mod files.
	existingTargetModArrFiles, err := filepath.Glob(filepath.Join(c.modDir, targetName+".*.mod"))
	if err != nil {
		return err
	}
	for _, f := range existingTargetModArrFiles {
		i, err := strconv.ParseInt(strings.Split(filepath.Base(f), ".")[1], 10, 64)
		if err != nil || int(i) >= len(versions) {
			if err := os.RemoveAll(f); err != nil {
				return err
			}
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
	pkgPath string,
	name string,
	version string,
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
	tmpModFile, err := createTmpModFileFromExisting(ctx, c.runner, outModFile, filepath.Join(c.modDir, name+".tmp.mod"))
	if err != nil {
		return errors.Wrap(err, "create tmp mod file")
	}
	defer errcapture.Close(&err, tmpModFile.Close, "close")

	runnable := c.runner.With(ctx, tmpModFile.Name(), c.modDir)
	if version != "" || tmpModFile.DirectPackage() == nil || c.update != runner.NoUpdatePolicy {
		// Steps 1 & 2: Resolve and download (if needed) thanks to 'go get' on our separate .mod file.
		targetPackage := module.Version{Path: pkgPath, Version: version}

		if tmpModFile.AutoReplaceDisabled() {
			if err := c.runner.With(ctx, tmpModFile.Name(), c.modDir).GetD(c.update, targetPackage.String()); err != nil {
				return errors.Wrap(err, "go get -d")
			}
		} else if err := autoReplaceGetD(ctx, c, tmpModFile, targetPackage); err != nil {
			return errors.Wrap(err, "go get -d")
		}
	}

	if err := merrors.New(
		tmpModFile.Reload(),
		tmpModFile.UpdateDirectPackage(pkgPath),
		tmpModFile.Flush(),
	).Err(); err != nil {
		return errors.Wrap(err, "updating direct package")
	}

	// Check if path is pointing to non-buildable package. Fail it is non-buildable. Hacky!
	if listOutput, err := runnable.List("-f={{.Name}}", pkgPath); err != nil {
		return err
	} else if !strings.HasSuffix(listOutput, "main") {
		return errors.Errorf("package %s is non-main (go list output %q), nothing to get and build", pkgPath, listOutput)
	}

	// We were working on tmp file, do atomic rename.
	if err := os.Rename(tmpModFile.Name(), outModFile); err != nil {
		return errors.Wrap(err, "rename")
	}

	// Step 3: Build and install.
	return c.runner.With(ctx, outModFile, c.modDir).Build(pkgPath, fmt.Sprintf("%s-%s", name, tmpModFile.DirectPackage().Version))
}

const modREADMEFmt = `# Project Development Dependencies.

This is directory which stores Go modules with pinned buildable package that is used within this repository, managed by https://github.com/bwplotka/bingo.

* Run ` + "`" + "bingo get" + "`" + ` to install all tools having each own module file in this directory.
* Run ` + "`" + "bingo get <tool>" + "`" + ` to install <tool> that have own module file in this directory.
* For Makefile: Make sure to put ` + "`" + "include %s/Variables.mk" + "`" + ` in your Makefile, then use $(<upper case tool name>) variable where <tool> is the %s/<tool>.mod.
* For shell: Run ` + "`" + "source %s/variables.env" + "`" + ` to source all environment variable for each tool.
* For go: Import ` + "`" + "%s/variables.go" + "`" + ` to for variable names.
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
		filepath.Join(relModDir, bingo.FakeRootModFileName),
		[]byte("module _ // Fake go.mod auto-created by 'bingo' for go -moddir compatibility with non-Go projects. Commit this file, together with other .mod files."),
		os.ModePerm,
	); err != nil {
		return err
	}

	// README.
	if err := ioutil.WriteFile(
		filepath.Join(relModDir, "README.md"),
		[]byte(fmt.Sprintf(modREADMEFmt, relModDir, relModDir, relModDir, relModDir)),
		os.ModePerm,
	); err != nil {
		return err
	}
	// gitignore.
	return ioutil.WriteFile(filepath.Join(relModDir, ".gitignore"), []byte(gitignore), os.ModePerm)
}

func createTmpModFileFromExisting(ctx context.Context, r *runner.Runner, modFile, tmpModFile string) (*bingo.ModFile, error) {
	if err := os.RemoveAll(tmpModFile); err != nil {
		return nil, errors.Wrap(err, "rm")
	}

	_, err := os.Stat(modFile)
	if err != nil && !os.IsNotExist(err) {
		return nil, errors.Wrapf(err, "stat module file %s", modFile)
	}
	if err != nil {
		if err := r.ModInit(ctx, filepath.Dir(modFile), tmpModFile, "_"); err != nil {
			return nil, errors.Wrap(err, "mod init")
		}
	} else {
		if err := copyFile(modFile, tmpModFile); err != nil {
			return nil, err
		}
	}
	return bingo.OpenModFile(tmpModFile)
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

// autoReplaceGetD runs 'go get -d' against separate go modules file with given arguments while making
// super replace statements are exactly the same as the target module.
// It's a very common case where modules mitigate faulty modules or conflicts with replace directives.
// Since we always download single tool dependency module per tool module, we can
// copy its replace if exists to fix this common case.
func autoReplaceGetD(ctx context.Context, c getConfig, tmpModFile *bingo.ModFile, targetPackage module.Version) (err error) {
	// Do initial get -d.
	gerr := c.runner.WithSilent(ctx, tmpModFile.Name(), c.modDir).GetD(c.update, targetPackage.String())

	runnable := c.runner.With(ctx, tmpModFile.Name(), c.modDir)

	// Regenerate replace statements.
	// Wrap all so we can do another get -d for clear output or error wrap.
	if err := func() error {
		if err := tmpModFile.Reload(); err != nil {
			return err
		}

		directModule := tmpModFile.DirectModule()
		if directModule == nil {
			if gerr == nil {
				return errors.Errorf("go get did not update the tmp module %v?", tmpModFile.Name())
			}
			// On error we might not have populated tmpModFile with module and version. In this case we need to trust get to put the
			// information what module was found (if any) in error message.
			foundVersionRe := fmt.Sprintf(`go: found %v in (\S*) (\S*)`, targetPackage.Path)

			// Try to match strings announcing what version was found (if we got to this stage).
			re, err := regexp.Compile(foundVersionRe)
			if err != nil {
				return errors.Wrapf(err, "regexp compile %v", foundVersionRe)
			}

			groups := re.FindAllStringSubmatch(gerr.Error(), 1)
			if len(groups) > 0 && len(groups[0]) > 2 {
				directModule = &module.Version{Path: groups[0][1], Version: groups[0][2]}
			} else {
				return errors.Errorf("we tried, but go get error was not helpful (our regexp: %v)", foundVersionRe)
			}
		}

		if err := tmpModFile.SetDirectRequire(&modfile.Require{
			Mod: *directModule,
		}); err != nil {
			return err
		}

		gopath, err := runnable.GoEnv("GOPATH")
		if err != nil {
			return errors.Wrap(err, "go env")
		}

		// We leverage fact that when go get fails because of version mismatch it actually does that after module is downloaded
		// with it's go mod file in the GOPATH/pkg/mod/....
		targetModFile := filepath.Join(gopath, "pkg", "mod", directModule.String(), "go.mod")
		targetModParsed, err := bingo.ParseModFileOrReader(targetModFile, nil)
		if err != nil {
			return errors.Wrapf(err, "parse target mod file %v", targetModFile)
		}

		// Set replace directives.
		if err := tmpModFile.SetReplace(targetModParsed.Replace...); err != nil {
			return err
		}
		return tmpModFile.Flush()
	}(); err != nil {
		return errors.Wrapf(runnable.GetD(c.update, targetPackage.String()), "also 'force' replace heal attempt failed %v", err)
	}
	return runnable.GetD(c.update, targetPackage.String())
}
