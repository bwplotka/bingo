// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/bwplotka/gobin/pkg/gobin"
	"github.com/bwplotka/gobin/pkg/gomodcmd"
	"github.com/pkg/errors"
)

// Get resolves and adds dependencies to the module under the -moddir and then builds and installs them in ${GOBIN}.
//
// Steps it performs:
//
// 1. Resolve which dependencies to add.
//
// Get uses `go get` underneath with support to following features;
//
// For each named package or package pattern, get must decide which version of the corresponding module to use.
//
// By default, get looks up the latest tagged release version, such as v0.4.5 or v1.2.3. If there are no tagged
// release versions, get looks up the latest tagged pre-release version, such as v0.0.1-pre1.
//
// If there are no tagged versions at all, get looks up the latest known commit. If the module is not already required
// at a later version (for example, a pre-release newer than the latest release), get will use the version it looked up.
// Otherwise, get will use the currently required version.
//
// This default version selection can be overridden by adding an @version suffix to the package argument, as in
// 'gobin get golang.org/x/text@v0.3.0'. The version may be a prefix: @v1 denotes the latest available version
// starting with v1. See 'go help modules' under the heading 'Module queries' for the full query syntax.
// For modules stored in source control repositories, the version suffix can also be a commit hash, branch identifier,
// or other syntax known to the source control system, as in 'gobin get golang.org/x/text@master'. Note that branches
// with names that overlap with other module query syntax cannot be selected explicitly. For example, the suffix @v2 means
// the latest version starting with v2, not the branch named v2.
//
// If a module under consideration is already a dependency of the current module under -moddir, then get will update the
// required version. Specifying a version earlier than the current required version is valid and downgrades the dependency.
// The version suffix @none indicates that the dependency should be removed entirely, downgrading or removing modules
// depending on it as needed.
//
// The version suffix @latest explicitly requests the latest minor release of the module named by the given path.
// The suffix @upgrade is like @latest but will not downgrade a module if it is already required at a revision or
// pre-release version newer than the latest released version. The suffix @patch requests the latest patch release:
// the latest released version with the same major and minor version numbers as the currently required version.
// Like @upgrade, @patch will not downgrade a module already required at a newer version. If the path is not already
// required, @upgrade and @patch are equivalent to @latest.
//
// Although get defaults to using the latest version of the module containing a named package, it does not use the
// latest version of that module's dependencies. Instead it prefers to use the specific dependency versions requested
// by that module. For example, if the latest A requires module B v1.2.3, while B v1.2.4 and v1.3.1 are also available,
// then 'gobin get A' will use the latest A but then use B v1.2.3, as requested by A. (If there are competing
// requirements for a particular module, then 'gobin get' resolves those requirements by taking the maximum requested version.)
//
// The -u flag instructs get to update modules providing dependencies of packages named on the command line to use newer minor
// or patch releases when available. Continuing the previous example, 'gobin get -u A' will use the latest A with B v1.3.1
// (not B v1.2.3). If B requires module C, but C does not provide any packages needed to build packages in A (not including
// tests), then C will not be updated.
//
// The -u=patch flag (not -u patch) also instructs get to update dependencies, but changes the default to select patch
// releases. Continuing the previous example, 'gobin get -u=patch A@latest' will use the latest A with B v1.2.4 (not B v1.2.3),
// while 'gobin get -u=patch A' will use a patch release of A instead.
//
// In general, adding a new dependency may require upgrading existing dependencies to keep a working build, and 'gobin get'
// does this automatically. Similarly, downgrading one dependency may require downgrading other dependencies, and 'gobin get' does this automatically as well.
//
// The -insecure flag permits fetching from repositories and resolving custom domains using insecure schemes such as HTTP. Use with caution.
//
//  2. Download (if needed),
//  3. Build, and install the named packages.
//
// If an argument names a module but not a package (because there is no Go source code in the module's root directory),
// then the error is returned (in opposite to go get, which skips the install step for that argument, instead of causing a build failure)
//
// Note that package patterns are allowed and are expanded after resolving the module versions. For example, 'gobin get golang.org/x/perf/cmd/...'
// adds the latest golang.org/x/perf and then installs the commands in that latest version.
//
// With no package arguments, 'go get' applies to Go package in the current directory, if any. In particular, 'gobin get -u' and 'gobin get -u=patch'
// update all the dependencies of that package. With no package arguments and also without -u, error is returned.
//https://www.amazon.co.uk/Acumobility-Level-Orange-Trigger-Point/dp/B07BHVFVZP/ref=pd_rhf_schuc_s_bmx_1_6/260-7448082-4381558?_encoding=UTF8&pd_rd_i=B07BHVFVZP&pd_rd_r=af543b92-75eb-4562-82cb-c17918def3e5&pd_rd_w=EaElB&pd_rd_wg=geQHA&pf_rd_p=f79852b8-e230-42d5-99d8-db2c761c54ac&pf_rd_r=38VNYYG6ZHC756H9V8KT&psc=1&refRID=38VNYYG6ZHC756H9V8KT
// For more about modules, see 'go help modules'.
//
//  4. Update moddir/binaries.go to make sure go mod preserves the given packages.
//
//  5. Runs go mod tidy.
//
// See also: gobin list.
func get(
	ctx context.Context,
	logger *log.Logger,
	r *gomodcmd.Runner,
	binFile string,
	update gomodcmd.GetUpdatePolicy,
	requestedPkgs ...string,
) (err error) {
	var nonVersionPackages []string
	if len(requestedPkgs) > 0 {
		// Steps 1 & 2: Resolve and download (if needed) thanks to go get on the separate go.mod file.
		if err := r.GetD(ctx, update, requestedPkgs...); err != nil {
			return err
		}

		for _, p := range requestedPkgs {
			s := strings.Split(p, "@")
			if len(s) > 1 && s[1] == "none" {
				continue
			}
			nonVersionPackages = append(nonVersionPackages, s[0])
		}

		// Step 3: Build and install. This will fail if any path is pointing to non-buildable package.
		if err := r.Install(ctx, nonVersionPackages...); err != nil {
			return err
		}
	}

	// Step 4: Regenerate moddir/binaries.go.
	f, err := os.OpenFile(binFile, os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "open %s", binFile)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			if err != nil {
				err = errors.Wrapf(err, "additionally error on close: %v", cerr)
				return
			}
			err = cerr
		}
	}()

	var pkgs []string
	if i, err := f.Stat(); err == nil {
		if i.Size() > 0 {
			pkgs, err = gobin.Parse(binFile, f)
			if err != nil {
				logger.Println("Parse error; file", binFile, "will be recreated. Err:", err)
			}
		}
	}
	if err := f.Truncate(0); err != nil {
		return errors.Wrapf(err, "truncate %s", binFile)
	}

	if _, err := f.Seek(0, 0); err != nil {
		return errors.Wrapf(err, "seek %s", binFile)
	}

	// DedupAndWrite will deduplicate and sort if needed.
	if err := gobin.DedupAndWrite(binFile, f, append(pkgs, nonVersionPackages...)); err != nil {
		return err
	}

	if len(requestedPkgs) == 0 {
		// Special mode of gobin, install all.
		if err := r.Install(ctx, pkgs...); err != nil {
			return err
		}
	}

	// Step 5: tidy.
	return r.ModTidy(ctx)
}
