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
	"github.com/pkg/errors"
	"golang.org/x/mod/modfile"
)

var goModVersionRegexp = regexp.MustCompile("^v[0-9]*$")

func parseTarget(rawTarget string) (name string, pkgPath string, versions []string, err error) {
	if rawTarget == "" {
		return "", "", nil, errors.New("target is empty, this should be filtered earlier")
	}

	s := strings.Split(rawTarget, "@")
	nameOrPackage := s[0]
	if len(s) > 1 {
		versions = strings.Split(s[1], ",")
	} else {
		versions = []string{""}
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

type installPackageConfig struct {
	runner    *runner.Runner
	modDir    string
	relModDir string
	update    runner.GetUpdatePolicy
}

type getConfig struct {
	runner    *runner.Runner
	modDir    string
	relModDir string
	update    runner.GetUpdatePolicy
	name      string
	rename    string
}

func (c getConfig) forPackage() installPackageConfig {
	return installPackageConfig{
		modDir:    c.modDir,
		relModDir: c.relModDir,
		runner:    c.runner,
		update:    c.update,
	}
}

func getAll(ctx context.Context, logger *log.Logger, c getConfig) (err error) {
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
		for i, targetPkg := range p.ToPackages() {
			if err := getPackage(ctx, logger, c.forPackage(), i, p.Name, targetPkg); err != nil {
				return errors.Wrapf(err, "%d: getting %s", i, targetPkg.String())
			}
		}
	}
	return nil
}

func existingModFiles(modDir string, targetName string) (existingModFiles []string, _ error) {
	existingModFiles, err := filepath.Glob(filepath.Join(modDir, targetName+".mod"))
	if err != nil {
		return nil, err
	}
	existingModArrFiles, err := filepath.Glob(filepath.Join(modDir, targetName+".*.mod"))
	if err != nil {
		return nil, err
	}
	return append(existingModFiles, existingModArrFiles...), nil
}

// get performs bingo get: it's like go get, but package aware, without go source files and on dedicated mod file.
// rawTarget is name or target package path, optionally with module version or array versions.
func get(ctx context.Context, logger *log.Logger, c getConfig, rawTarget string) (err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute) // TODO(bwplotka): Put as param?
	defer cancel()

	if err := ensureModDirExists(logger, c.relModDir); err != nil {
		return errors.Wrap(err, "ensure mod dir")
	}

	if rawTarget == "" {
		// Empty target means to get all. It recursively invokes get for each existing binary.
		return getAll(ctx, logger, c)
	}

	name, pkgPath, versions, err := parseTarget(rawTarget)
	if err != nil {
		return errors.Wrapf(err, "parse %v", rawTarget)
	}

	if c.rename != "" {
		if versions[0] != "" || len(versions) > 1 {
			return errors.Errorf("rename cannot take version arguments (string after @), got %v", versions)
		}
		if err := validateNewName(versions, c.modDir, name, c.rename); err != nil {
			return errors.Wrap(err, "-n")
		}
	}

	targetName := name
	if c.name != "" {
		if err := validateNewName(versions, c.modDir, name, c.name); err != nil {
			return errors.Wrap(err, "-n")
		}
		targetName = c.name
	}

	if versions[0] == "none" {
		// none means we no longer want to version this package.
		// NOTE: We don't remove binaries.
		return removeAllGlob(filepath.Join(c.modDir, name+".*"))
	}

	existing, err := existingModFiles(c.modDir, targetName)
	if err != nil {
		return errors.Wrapf(err, "existing mod files for %v", targetName)
	}
	targets := make([]bingo.Package, 0, len(versions))
	for i, v := range versions {
		if len(existing) < i {
			e := existing[i]

			mf, err := bingo.OpenModFile(e)
			if err != nil {
				return errors.Wrapf(err, "found unparsable mod file %v. Uninstall it first via get %v@none or fix it manually.", e, targetName)
			}
			defer errcapture.Close(&err, mf.Close, "close")

			if mf.DirectPackage() != nil {
				if pkgPath != "" && pkgPath != mf.DirectPackage().Path() {
					// TODO(bwplotka): Sketchy.
					return errors.Errorf("failed to install %q under %q name as binary with the same name is already installed for path %q. "+
						"Uninstall existing tool using `%v@none` or use `-n` flag to choose different name", pkgPath, targetName, mf.DirectPackage().Path(), targetName)
				}
				p := *mf.DirectPackage()
				if v != "" {
					p.Module.Version = v
				}
				targets = append(targets, p)
				continue

			} else if c.rename != "" {
				return errors.Errorf("bingo tool module %v.mod is malformed; can't get package path and version for"+
					" base tool. Use @none to uninstall or reinstall base package; err: %v\n", targetName, err)
			}
		} else if c.rename != "" {
			return errors.Errorf("nothing to rename, no module %v.mod was found; can't get package path and version for"+
				" base tool. Install base package first; err: %v\n", targetName, err)
		}
		p := bingo.Package{RelPath: pkgPath}
		if v != "" {
			p.Module.Version = versions[i]
		}
		targets = append(targets, p)
	}

	if c.rename != "" {
		targetName = c.rename
	}

	for _, t := range targets {
		if err := getPackage(ctx, logger, c.forPackage(), 0, targetName, t); err != nil {
			return errors.Wrapf(err, "%s.mod: getting %s", targetName, t)
		}
	}

	// Rename requested. Remove old mod(s) in this case, but only at the end.
	if c.rename != "" && targetName != name {
		defer func() { _ = removeAllGlob(filepath.Join(c.modDir, name+".*")) }()
	}

	// Remove target unused arr mod files based on version file.
	existingTargetModArrFiles, gerr := filepath.Glob(filepath.Join(c.modDir, targetName+".*.mod"))
	if gerr != nil {
		err = gerr
		return
	}
	for _, f := range existingTargetModArrFiles {
		i, perr := strconv.ParseInt(strings.Split(filepath.Base(f), ".")[1], 10, 64)
		if perr != nil || int(i) >= len(versions) {
			if rerr := os.RemoveAll(f); rerr != nil {
				err = rerr
				return
			}
		}
	}
	return nil
}

func validateNewName(versions []string, modDir, old, new string) error {
	if new == old {
		return errors.Errorf("cannot be the same as module name %v", new)
	}
	if versions[0] == "none" {
		return errors.Errorf("cannot use with @none logic")
	}
	newExisting, err := existingModFiles(modDir, new)
	if err != nil {
		return errors.Wrapf(err, "existing mod files for %v", new)
	}
	if len(newExisting) > 0 {
		return errors.Errorf("found existing installed binaries %v under name you want to rename on. Remove target name %s or use different one", newExisting, new)
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

func validateTargetName(targetName string) error {
	if targetName == "cmd" {
		return errors.Errorf("package would be installed with ambiguous name %s. This is a common, but slightly annoying package layout"+
			"It's advised to choose unique name with -n flag", targetName)
	}
	if targetName == strings.TrimSuffix(bingo.FakeRootModFileName, ".mod") {
		return errors.Errorf("requested binary with name %q`. This is impossible, choose different name using -n flag", strings.TrimSuffix(bingo.FakeRootModFileName, ".mod"))
	}
	return nil
}

func updateModAndVersionFromGoGetOutput(runnable runner.Runnable, update runner.GetUpdatePolicy, target *bingo.Package) (err error) {
	// Do initial go get -d. If it errors out, we rely on output to find the latest target version.
	out, gerr := runnable.GetD(update, target.String())

	// Wrap all with runnable output.
	defer func() {
		if err != nil {
			if gerr != nil {
				out = errors.Wrap(gerr, out).Error()
			}
			err = errors.Wrapf(err, "resolve; go get -d output: %v", out)
		}
	}()

	// TODO(bwplotka) Obviously hacky but reliable so far.
	// Try to match strings announcing what version was found (if we got to this stage).
	downloadingRe := fmt.Sprintf(`go: downloading (%v) (\S*)`, target.Path())
	upgradeRe := `go: (\S*) upgrade => (\S*)`
	foundVersionRe := fmt.Sprintf(`go: found %v in (\S*) (\S*)`, target.Path())

	re, err := regexp.Compile(downloadingRe)
	if err != nil {
		return errors.Wrapf(err, "regexp compile %v", downloadingRe)
	}
	if !re.MatchString(out) {
		re = regexp.MustCompile(upgradeRe)
		if !re.MatchString(out) {
			re, err = regexp.Compile(foundVersionRe)
			if err != nil {
				return errors.Wrapf(err, "regexp compile %v", foundVersionRe)
			}
		}
	}

	groups := re.FindAllStringSubmatch(out, 1)
	if len(groups) == 0 || len(groups[0]) < 3 {
		return errors.Errorf("go get did not found the package (or our regexps did not match: %v)", []string{
			downloadingRe,
			upgradeRe,
			foundVersionRe,
		})
	}

	target.RelPath, err = filepath.Rel(groups[0][1], target.RelPath)
	if err != nil {
		return errors.Wrap(err, "rel")
	}

	// Update target with old version and module.
	target.Module.Path = groups[0][1]
	target.Module.Version = groups[0][2]
	return nil
}

// getPackage takes package array index, tool name and package path (also module path and version which are optional) and
// generates new module with the given package's module as the only dependency (direct require statement).
// For generation purposes we take the existing <name>.mod file (if exists, if paths matches). This allows:
//  * Comments to be preserved.
//  * First direct require module will be preserved (unless version changes)
//  * Replace to be preserved if the // bingo:no_replace_fetch commend is found it such mod file.
// As resolution of module vs package for Go Module is convoluted and all code is under internal dir, we have to rely on `go` binary
// capabilities and output.
// TODO(bwplotka): Consider copying code for it? Of course it's would be easier if such tool would exist in Go project itself (:
func getPackage(ctx context.Context, logger *log.Logger, c installPackageConfig, i int, name string, target bingo.Package) (err error) {
	// Cleanup all bingo modules' tmp files for fresh start.
	if err := cleanGoGetTmpFiles(c.modDir); err != nil {
		return err
	}

	// The out module file we generate/maintain keep in modDir.
	outModFile := filepath.Join(c.modDir, name+".mod")
	if i > 0 {
		// Handle array go modules.
		outModFile = filepath.Join(c.modDir, fmt.Sprintf("%s.%d.mod", name, i))
	}

	// If we don't have all information or update is set, resolve version.
	var replaceStmts []*modfile.Replace
	if target.Module.Version == "" || target.Module.Path == "" || c.update != runner.NoUpdatePolicy {
		// Set up totally empty mod file to get clear version to install.
		tmpEmptyModFile, err := createTmpModFileFromExisting(ctx, c.runner, logger, "", filepath.Join(c.modDir, name+"-e.tmp.mod"))
		if err != nil {
			return errors.Wrap(err, "create empty tmp mod file")
		}
		defer errcapture.Close(&err, tmpEmptyModFile.Close, "close")

		runnable := c.runner.With(ctx, tmpEmptyModFile.Name(), c.modDir)
		if err := updateModAndVersionFromGoGetOutput(runnable, c.update, &target); err != nil {
			return err
		}

		// autoReplace is reproducing replace statements to be exactly the same as the target module we want to install.
		// It's a very common case where modules mitigate faulty modules or conflicts with replace directives.
		// Since we always download single tool dependency module per tool module, we can
		// copy its replace if exists to fix this common case.
		gopath, err := runnable.GoEnv("GOPATH")
		if err != nil {
			return errors.Wrap(err, "go env")
		}

		// We leverage fact that when go get runs if downloads the version we find as relevant locally
		// in the GOPATH/pkg/mod/...
		targetModFile := filepath.Join(gopath, "pkg", "mod", target.Module.String(), "go.mod")
		targetModParsed, err := bingo.ParseModFileOrReader(targetModFile, nil)
		if err != nil {
			return errors.Wrapf(err, "parse target mod file %v", targetModFile)
		}

		// Store replace to auto-update if needed.
		replaceStmts = targetModParsed.Replace
	}

	// Now we should have target with all required info, prepare tmp file.
	if err := cleanGoGetTmpFiles(c.modDir); err != nil {
		return err
	}
	tmpModFile, err := createTmpModFileFromExisting(ctx, c.runner, logger, outModFile, filepath.Join(c.modDir, name+".tmp.mod"))
	if err != nil {
		return errors.Wrap(err, "create tmp mod file")
	}
	defer errcapture.Close(&err, tmpModFile.Close, "close")

	if !tmpModFile.AutoReplaceDisabled() && len(replaceStmts) > 0 {
		if err := tmpModFile.SetReplace(replaceStmts...); err != nil {
			return err
		}
	}

	if err := tmpModFile.SetDirectRequire(target); err != nil {
		return err
	}

	if err := tmpModFile.Flush(); err != nil {
		return err
	}

	runnable := c.runner.With(ctx, tmpModFile.Name(), c.modDir)
	if err := install(runnable, tmpModFile.Name(), tmpModFile.DirectPackage()); err != nil {
		return errors.Wrap(err, "install")
	}

	// We were working on tmp file, do atomic rename.
	if err := os.Rename(tmpModFile.Name(), outModFile); err != nil {
		return errors.Wrap(err, "rename")
	}
	return nil
}

func install(runnable runner.Runnable, name string, pkg *bingo.Package) (err error) {
	if err := validateTargetName(name); err != nil {
		return errors.Wrap(err, pkg.String())
	}

	// Check if path is pointing to non-buildable package. Fail it is non-buildable. Hacky!
	if listOutput, err := runnable.List("-f={{.Name}}", pkg.Path()); err != nil {
		return err
	} else if !strings.HasSuffix(listOutput, "main") {
		return errors.Errorf("package %s is non-main (go list output %q), nothing to get and build", pkg.Path(), listOutput)
	}
	return runnable.Build(pkg.Path(), fmt.Sprintf("%s-%s", name, pkg.Module.Version))
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

func createTmpModFileFromExisting(ctx context.Context, r *runner.Runner, logger *log.Logger, modFile, tmpModFile string) (*bingo.ModFile, error) {
	if err := os.RemoveAll(tmpModFile); err != nil {
		return nil, errors.Wrap(err, "rm")
	}

	if modFile != "" {
		_, err := os.Stat(modFile)
		if err != nil && !os.IsNotExist(err) {
			return nil, errors.Wrapf(err, "stat module file %s", modFile)
		}
		if err == nil {
			// Only use existing mod file on successful parse.
			o, err := bingo.OpenModFile(modFile)
			if err == nil {
				if err := o.Close(); err != nil {
					return nil, err
				}
				if err := copyFile(modFile, tmpModFile); err != nil {
					return nil, err
				}
				return bingo.OpenModFile(tmpModFile)
			}
			logger.Printf("bingo tool module %v is malformed; it will be recreated; err: %v\n", modFile, err)
		}
	}

	// Create from scratch.
	if err := r.ModInit(ctx, filepath.Dir(modFile), tmpModFile, "_"); err != nil {
		return nil, errors.Wrap(err, "mod init")
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
