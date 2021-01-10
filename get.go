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
}

func (c getConfig)  forPackage() getPackageConfig {
	return getPackageConfig{
		modDir:    c.modDir,
		relModDir: c.relModDir,
		update:    c.update,
		runner:    c.runner,
	}
}


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

func getAll(ctx context.Context,logger *log.Logger,c getConfig) (err error) {
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
	if rawTarget == "" {
		// Empty means get all. It recursively invokes get for each existing binary.
		return getAll(ctx, logger, c)
	}

	name, pkgPath, versions, err := parseTarget(rawTarget)
	if err != nil {
		return errors.Wrapf(err, "parse %v", rawTarget)
	}

	if c.rename != "" {
		if c.rename == name {
			return errors.Wrapf(err, "-r cannot be the same as module name %v", c.rename)
		}
		renamedExisting, err := existingModFiles(c.modDir, c.rename)
		if err != nil {
			return errors.Wrapf(err, "existing mod files for %v", c.rename)
		}

		if len(renamedExisting) > 0 {
			return errors.Errorf("found existing installed binaries %v under name you want to rename on. Remove target name %s or use different one", renamedExisting, c.rename)
		}
	}

	targetName := name
	if c.name != "" {
		if c.name == name{
			return errors.Wrapf(err, "-n cannot be the same as module name %v", c.name)
		}
		if versions[0] == "none" {
			return errors.Errorf("cannot use -n option with @none logic")
		}
		targetName = c.name
	}

	if versions[0] == "none" {
		// none means we no longer want to version this package.
		// NOTE: We don't remove binaries.
		return removeAllGlob(filepath.Join(c.modDir, targetName+".*"))
	}

	defer func() {
		if err == nil {
			return
		}

		// Rename requested. Remove old mod(s) in this case, but only at the end.
		if	c.rename != "" && targetName != name {
			defer func() {_ = removeAllGlob(filepath.Join(c.modDir, name+".*")) }()
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
	}()

	existing, err := existingModFiles(c.modDir, targetName)
	if err != nil {
		return errors.Wrapf(err, "existing mod files for %v", targetName)
	}

	if len(versions) == 1 {
		// Single version mode.
		if len(existing) > 0 {
			existingPkg, err := bingo.ModDirectPackage(existing[0])
			if err == nil {
				// Check if existing mod file is not referencing totally different path.
				if pkgPath != "" && pkgPath != existingPkg.Path() {
					return errors.Errorf("failed to install %q under %q name as binary with the same name is already installed for path %q. "+
						"Uninstall existing %q tool using `@none` or use `-n` flag to choose different name", pkgPath, targetName, existingPkg.Path(), targetName)
				}

				if versions[0] != "" {
					existingPkg.Module.Version = versions[0]
				}

				if c.rename != "" {
					targetName = c.rename
				}

				if err := getPackage(ctx, logger, c.forPackage(), 0, targetName, existingPkg); err != nil {
					return errors.Wrapf(err, "%s.mod: getting %s", targetName, existingPkg)
				}

				// Done for this case.
				return nil
			}

			// Malformed modfile or missing info about existing direct package.
			if pkgPath != "" {
				return errors.Wrapf(err, "bingo tool module %v is malformed. Use full package name to reinstall it", existing[0])
			}

			if c.rename != "" {
				return errors.Errorf("bingo tool module %v.mod is malformed; can't rename broken tool. Use @none of full package if  it will be recreated; err: %v\n", targetName, err)
			}

			logger.Printf("bingo tool module %v.mod is malformed; it will be recreated; err: %v\n", targetName, err)
		}

		// Non array (that could be array before).
		// Get full import path from any existing module file for this name.
		if pkgPath == "" && len(existing) == 0 {
			// Binary referenced by name, get full package name if module file exists.
			return errors.Errorf("binary %q was not installed before. Use full package name to install it", targetName)
		}




		if len(versions) == 0 {
			for _, f := range existing {
				pkg, err := bingo.ModDirectPackage(f)
				if err != nil {
					return errors.Wrapf(err, "binary %q was installed, but go module %s is malformed. Use full package name to reinstall it", name, f)
				}
				versions = append(versions, pkg.Module.Version)
			}

			if len(versions) == 0 {
				// Binary never seen before and requested to be installed from latest.
				versions = append(versions, "")
			}
		}

		for i, version := range versions {
			// Ideally we can pa
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
	}

	for _, f := range existing {
		existingPkg, err := bingo.ModDirectPackageOrNil(f)
		if err != nil {
			if pkgPath != "" {
				return errors.Wrapf(err, "bingo tool module %v.mod is malformed. but go modules %s is malformed. Use full package name to reinstall it", targetName, f)
			}
			// This will be handled by getPackage correctly (existing file replaced atomically).
		}

		if existingPkg != nil {
			// Check if existing mod file is not referencing totally different path.
			if pkgPath != "" && pkgPath != existingPkg.Path() {
				return errors.Errorf("failed to install %q under %q name as binary with the same name is already installed for path %q. "+
					"Uninstall existing %q tool using `@none` or use `-n` flag to choose different name", pkgPath, targetName, existingPkg.Path(), targetName)
			}
			pkgPath = existingPkg.Path()
		}
	}

	// Get full import path from any existing module file for this name.
	if pkgPath == "" && len(existing) == 0 {
		// Binary referenced by name, get full package name if module file exists.
		return errors.Errorf("binary %q was not installed before. Use full package name to install it", targetName)
	}



	if len(versions) == 0 {
		for _, f := range existing {
			pkg, err := bingo.ModDirectPackage(f)
			if err != nil {
				return errors.Wrapf(err, "binary %q was installed, but go module %s is malformed. Use full package name to reinstall it", name, f)
			}
			versions = append(versions, pkg.Module.Version)
		}

		if len(versions) == 0 {
			// Binary never seen before and requested to be installed from latest.
			versions = append(versions, "")
		}
	}

	if c.rename != "" {
		targetName = c.rename

		renamedExisting, err := existingModFiles(c.modDir, targetName)
		if err != nil {
			return errors.Wrapf(err, "existing mod files for %v", targetName)
		}

		if len(renamedExisting) > 0 {
			return errors.Errorf("found existing installed binaries %v under name you want to rename on. Remove target name %s or use -n to choose different one", existing, c.rename)
		}
	}

	for i, version := range versions {
		// Ideally we can pa
		if err := getOne(ctx, logger, c, i, pkgPath, targetName, version); err != nil {
			return errors.Wrapf(err, "%d: getting %s", i, version)
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

type getPackageConfig struct {
	runner    *runner.Runner
	modDir    string
	relModDir string
	update    runner.GetUpdatePolicy
}

func validateTargetName(targetName string) error {
	if targetName == "cmd" {
		return errors.Errorf("package would be installed with ambiguous name %s. This is a common, but slightly annoying package layout"+
			"It's advised to choose unique name with -n flag",targetName)
	}
	if targetName == strings.TrimSuffix(bingo.FakeRootModFileName, ".mod") {
		return errors.Errorf("requested binary with name %q`. This is impossible, choose different name using -n flag", strings.TrimSuffix(bingo.FakeRootModFileName, ".mod"))
	}
	return nil
}

// getPackage takes package array index, module and package path, tool name and (optionally) version and generates new module with the given package's module as the only
// require statement. For generation purposes we take the existing <name>.mod file (if exists, if paths matches).
// This allows:
//  * Comments to be preserved.
//  * First direct require module will be preserved.
//  * Replace to be preserved if the // bingo:no_replace_fetch commend is found it such mod file.
// As resolution of module vs package for Go Module is convoluted and all code is under internal dir, we have to rely on `go` binary
// capabilities.
// TODO(bwplotka): Consider copying code for it? Of course it's would be easier if such tool would exist in Go project itself (:
func getPackage(ctx context.Context, logger *log.Logger, c getPackageConfig, i int, name string, target bingo.Package) (err error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	if err := validateTargetName(name); err != nil {
		return errors.Wrap(err, target.String())
	}

	// The out module file we generate/maintain keep in modDir.
	outModFile := filepath.Join(c.modDir, name+".mod")
	if i > 0 {
		// Handle array go modules.
		outModFile = filepath.Join(c.modDir, fmt.Sprintf("%s.%d.mod", name, i))
	}

	// Cleanup all bingo modules' tmp files for fresh start.
	if err := cleanGoGetTmpFiles(c.modDir); err != nil {
		return err
	}
	if err := ensureModDirExists(logger, c.relModDir); err != nil {
		return errors.Wrap(err, "ensure mod dir")
	}

	// Set up tmp file that we will work on for now. This is to avoid partial updates.
	tmpModFile, err := createTmpModFileFromExisting(ctx, c.runner, logger, outModFile, filepath.Join(c.modDir, name+".tmp.mod"))
	if err != nil {
		return errors.Wrap(err, "create tmp mod file")
	}
	defer errcapture.Close(&err, tmpModFile.Close, "close")

	if tmpModFile.DirectPackage()


	runnable := c.runner.With(ctx, tmpModFile.Name(), c.modDir)
	targetPackage := module.Version{Path: pkgPath, Version: version}

	// Try to resolve and re-download only if needed.
	if version != "" || tmpModFile.DirectPackage() == nil ||
		tmpModFile.DirectPackage().String() != targetPackage.String() ||
		c.update != runner.NoUpdatePolicy {

		// Steps 1 & 2: Resolve and download thanks to 'go get' on our separate .mod file.
		if !tmpModFile.AutoReplaceDisabled() {

			if err := autoReplace(ctx, c, tmpModFile, targetPackage); err != nil {
				return errors.Wrap(err, "go get -d")
			}
		}

		out, err := runnable.GetD(c.update, targetPackage.String())
		if err != nil {
			return errors.Wrapf(err, "go get -d: %v", out)
		}
		fmt.Println(out)

		// We need to reload due to potential get -d changes.
		if err := merrors.New(tmpModFile.Reload(), tmpModFile.UpdateDirectPackage(pkgPath), tmpModFile.Flush()).Err(); err != nil {
			return errors.Wrap(err, "updating direct package")
		}
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

func createTmpModFileFromExisting(ctx context.Context, r *runner.Runner,logger *log.Logger, modFile, tmpModFile string) (*bingo.ModFile, error) {
	if err := os.RemoveAll(tmpModFile); err != nil {
		return nil, errors.Wrap(err, "rm")
	}

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

// autoReplace runs 'go get -d' against separate go modules file with given arguments while making
// super replace statements are exactly the same as the target module.
// It's a very common case where modules mitigate faulty modules or conflicts with replace directives.
// Since we always download single tool dependency module per tool module, we can
// copy its replace if exists to fix this common case.
func autoReplace(ctx context.Context, c getConfig, tmpModFile *bingo.ModFile, targetPackage module.Version) (err error) {
	runnable := c.runner.With(ctx, tmpModFile.Name(), c.modDir)

	// Do initial get -d. No matter if it succeeds or not, we need to take the target version it tried to ensure we
	// have correct replace directive.
	out, gerr := runnable.GetD(c.update, targetPackage.String())

	// Regenerate replace statements.
	// Wrap all with gerr
	if err := func() error {
		if err := tmpModFile.Reload(); err != nil {
			return err
		}

		// On error we might not have populated tmpModFile with module and version. In this case we need to trust get to put the
		// information what module was found (if any) in error message.
		foundVersionRe := fmt.Sprintf(`go: found %v in (\S*) (\S*)`, targetPackage.Path)

		// Try to match strings announcing what version was found (if we got to this stage).
		re, err := regexp.Compile(foundVersionRe)
		if err != nil {
			return errors.Wrapf(err, "regexp compile %v", foundVersionRe)
		}

		groups := re.FindAllStringSubmatch(out, 1)
		if len(groups) == 0 || len(groups[0]) < 3 {
			return errors.Errorf("we tried, but go get error was not helpful (our regexp: %v)", foundVersionRe)
		}

		//targetDirectModule = &module.Version{Path: groups[0][1], Version: groups[0][2]}

		directModule := tmpModFile.DirectModule()
		if directModule != nil {

		}

		if err := tmpModFile.SetDirectRequire(&modfile.Require{Mod: *directModule}); err != nil {
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
		return errors.Wrapf(gerr, "also 'force' replace heal attempt failed %v", err)
	}
	return nil
}
