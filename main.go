// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"text/tabwriter"

	"github.com/bwplotka/bingo/pkg/bingo"
	"github.com/bwplotka/bingo/pkg/runner"
	"github.com/bwplotka/bingo/pkg/version"
	"github.com/oklog/run"
	"github.com/pkg/errors"
)

func exitOnUsageError(usage func(), v ...interface{}) {
	fmt.Println(append([]interface{}{"Error:"}, v...)...)
	fmt.Println()
	usage()
	os.Exit(1)
}

func main() {
	logger := log.New(os.Stderr, "", 0)

	// Main flags.
	flags := flag.NewFlagSet("bingo", flag.ContinueOnError)
	verbose := flags.Bool("v", false, "Print more'")
	help := flags.Bool("h", false, "Print usage and exit.")

	// Get flags.
	getFlags := flag.NewFlagSet("bingo get", flag.ContinueOnError)
	getModDir := getFlags.String("moddir", ".bingo", "Directory where separate modules for each binary will be"+
		" maintained. Feel free to commit this directory to your VCS to bond binary versions to your project code. If the directory"+
		" does not exist bingo logs and assumes a fresh project.")
	getName := getFlags.String("n", "", "The -n flag instructs to get binary and name it with given name instead of default,"+
		" so the last element of package directory. Allowed characters [A-z0-9._-]. If -n is used and no package/binary is specified,"+
		" bingo get will return error. If -n is used with existing binary name, copy of this binary will be done. Cannot be used with -r")
	getRename := getFlags.String("r", "", "The -r flag instructs to get existing binary and rename it with given name."+
		" Allowed characters [A-z0-9._-]. If -r is used and no package/binary is specified or non existing binary name is used, bingo"+
		" will return error. Cannot be used with -n.")
	goCmd := getFlags.String("go", "go", "Path to the go command.")
	getUpdate := getFlags.Bool("u", false, "The -u flag instructs get to update modules providing dependencies of packages named on the command line to use newer minor or patch releases when available.")
	getUpdatePatch := getFlags.Bool("upatch", false, "The -upatch flag (not -u patch) also instructs get to update dependencies, but changes the default to select patch releases.")
	getInsecure := getFlags.Bool("insecure", false, "Use -insecure flag when using 'go get'")

	// Go flags is so broken, need to add shadow -v flag to make those work in both before and after `get` command.
	getVerbose := getFlags.Bool("v", false, "Print more'")

	// List flags.
	listFlags := flag.NewFlagSet("bingo list", flag.ContinueOnError)
	listModDir := listFlags.String("moddir", ".bingo", "Directory where separate modules for each binary is"+
		" maintained. If does not exists, bingo list will fail.")
	// Go flags is so broken, need to add shadow -v flag to make those work in both before and after `list` command.
	listVerbose := listFlags.Bool("v", false, "Print more'")

	flags.Usage = func() {
		getFlagsHelp := &strings.Builder{}
		getFlags.SetOutput(getFlagsHelp)
		getFlags.PrintDefaults()

		listFlagsHelp := &strings.Builder{}
		listFlags.SetOutput(listFlagsHelp)
		listFlags.PrintDefaults()
		fmt.Printf(bingoHelpFmt, getFlagsHelp.String(), listFlagsHelp.String())
	}
	if err := flags.Parse(os.Args[1:]); err != nil {
		exitOnUsageError(flags.Usage, "Failed to parse flags:", err)
	}

	if *help {
		flags.Usage()
		os.Exit(0)
	}

	if flags.NArg() == 0 {
		exitOnUsageError(flags.Usage, "No command specified")
	}
	var cmdFunc func(ctx context.Context, r *runner.Runner) error
	switch flags.Arg(0) {
	case "get":
		getFlags.SetOutput(os.Stdout)
		if err := getFlags.Parse(flags.Args()[1:]); err != nil {
			exitOnUsageError(flags.Usage, "Failed to parse flags for get command:", err)
		}

		if !*verbose && *getVerbose {
			*verbose = true
		}

		if *getModDir == "" {
			exitOnUsageError(flags.Usage, "'moddir' flag cannot be empty")
		}

		if *goCmd == "" {
			exitOnUsageError(flags.Usage, "'go' flag cannot be empty")
		}

		upPolicy := runner.NoUpdatePolicy
		if *getUpdate {
			upPolicy = runner.UpdatePolicy
		}
		if *getUpdatePatch {
			upPolicy = runner.UpdatePatchPolicy
		}

		if getFlags.NArg() > 1 {
			exitOnUsageError(flags.Usage, "Too many arguments except none or binary/package ")
		}

		target := getFlags.Arg(0)
		if *getRename != "" && *getName != "" {
			exitOnUsageError(flags.Usage, "Both -n and -r were specified. You can either rename or create new one.")
		}
		if *getName != "" && !regexp.MustCompile(`[a-zA-Z0-9.-_]+`).MatchString(*getName) {
			exitOnUsageError(flags.Usage, *getName, "-n name contains not allowed characters")
		}
		if *getRename != "" && !regexp.MustCompile(`[a-zA-Z0-9.-_]+`).MatchString(*getRename) {
			exitOnUsageError(flags.Usage, *getRename, "-r name contains not allowed characters")
		}

		cmdFunc = func(ctx context.Context, r *runner.Runner) (err error) {
			relModDir := *getModDir
			modDir, err := filepath.Abs(relModDir)
			if err != nil {
				return errors.Wrap(err, "abs")
			}
			defer func() {
				if err != nil {
					// Leave tmp files on error for debug purposes.
					_ = cleanGoGetTmpFiles(modDir)
				}
			}()

			cfg := getConfig{
				runner:    r,
				modDir:    modDir,
				relModDir: relModDir,
				update:    upPolicy,
				name:      *getName,
				rename:    *getRename,
				verbose:   *verbose,
			}

			if err := get(ctx, logger, cfg, target); err != nil {
				return errors.Wrap(err, "get")
			}

			pkgs, err := bingo.ListPinnedMainPackages(logger, modDir, true)
			if err != nil {
				return errors.Wrap(err, "list pinned")
			}
			if len(pkgs) == 0 {
				return bingo.RemoveHelpers(modDir)
			}
			return bingo.GenHelpers(relModDir, version.Version, pkgs)
		}
	case "list":
		listFlags.SetOutput(os.Stdout)
		if err := listFlags.Parse(flags.Args()[1:]); err != nil {
			exitOnUsageError(flags.Usage, "Failed to parse flags for get command:", err)
		}

		if !*verbose && *listVerbose {
			*verbose = true
		}

		if *listModDir == "" {
			exitOnUsageError(flags.Usage, "'moddir' flag cannot be empty")
		}

		if listFlags.NArg() > 1 {
			exitOnUsageError(flags.Usage, "Too many arguments; only one binary/package or no argument is expected ")
		}

		target := listFlags.Arg(0)
		cmdFunc = func(ctx context.Context, r *runner.Runner) error {
			modDir, err := filepath.Abs(*listModDir)
			if err != nil {
				return errors.Wrap(err, "abs")
			}
			pkgs, err := bingo.ListPinnedMainPackages(logger, modDir, false)
			if err != nil {
				return err
			}

			w := new(tabwriter.Writer)
			w.Init(os.Stdout, 4, 4, 1, '\t', 0)
			defer func() { _ = w.Flush() }()

			_, _ = fmt.Fprintf(w, "Name\tBinary Name\tPackage @ Version")
			_, _ = fmt.Fprintf(w, "\n----\t-----------\t-----------------")
			for _, p := range pkgs {
				if target != "" && p.Name != target {
					continue
				}
				for _, v := range p.Versions {
					_, _ = fmt.Fprintf(w, "\n%s\t%s-%s\t%s@%s", p.Name, p.Name, v.Version, p.PackagePath, v.Version)
				}
				if target != "" {
					return nil
				}
			}

			if target != "" {
				return errors.Errorf("Pinned tool %s not found", target)
			}
			return nil
		}
	case "Version":
		cmdFunc = func(ctx context.Context, r *runner.Runner) error {
			_, err := fmt.Fprintln(os.Stdout, version.Version)
			return err
		}
	default:
		exitOnUsageError(flags.Usage, "No such command", flags.Arg(0))
	}

	g := &run.Group{}
	g.Add(run.SignalHandler(context.Background(), syscall.SIGINT, syscall.SIGTERM))

	// Command run actor.
	{
		ctx, cancel := context.WithCancel(context.Background())
		g.Add(func() error {
			r, err := runner.NewRunner(ctx, *getInsecure, *goCmd)
			if err != nil {
				return err
			}

			if *verbose {
				r.Verbose()
			}
			return cmdFunc(ctx, r)
		}, func(error) {
			cancel()
		})
	}
	if err := g.Run(); err != nil {
		if *verbose {
			// Use %+v for github.com/pkg/errors error to print with stack.
			logger.Fatalf("Error: %+v", errors.Wrapf(err, "%s command failed", flags.Arg(0)))
		}
		logger.Fatalf("Error: %v", errors.Wrapf(err, "%s command failed", flags.Arg(0)))
	}
}

const bingoHelpFmt = `bingo: 'go get' like, simple CLI that allows automated versioning of Go package level binaries (e.g required as dev tools by your project!)
built on top of Go Modules, allowing reproducible dev environments. 'bingo' allows to easily maintain a separate, nested Go Module for each binary.

For detailed examples see: https://github.com/bwplotka/bingo

'bingo' supports following commands:

Commands:

  get <flags> [<package or binary>[@version1 or none,version2,version3...]]

Similar to 'go get' you can pull, install and pin required 'main' (buildable Go) package as your tool in your project.

'bingo get <repo/org/tool>' will resolve given main package path, download it using 'go get -d', then will produce directory (controlled by -moddir flag) and put
separate, specially commented module called <tool>.mod. After that, it installs given package as '$GOBIN/<tool>-<Version>'.

Once installed at least once, 'get' allows to reference the tool via it's name (without Version) to install, downgrade, upgrade or remove.
Similar to 'go get' you can get binary with given Version: a git commit, git tag or Go Modules pseudo Version after @:

'bingo get <repo/org/tool>@<Version>' or 'bingo get <tool>@<Version>'

'get' without any argument will download and get ALL the tools in the moddir directory.
'get' also allows bulk pinning and install. Just specify multiple versions after '@':

'bingo get <tool>@<version1,version2,tag3>'

Similar to 'go get' you can use -u and -u=patch to control update logic and '@none' to remove binary.

Once pinned apart of 'bingo get', you can also use 'go build -modfile .bingo/<tool>.mod -o=<where you want to build> <tool package>' to install
correct Version of a tool.

Note that 'bingo' creates additional useful files inside -moddir:

* '<moddir>/Variables.mk': When included in your Makefile ('include <moddir>/Variables.mk'), you can refer to each binary
using '$(TOOL)' variable. It will also  install correct Version if missing.
* '<moddir>/variables.env': When sourced ('source <moddir>/variables.env') you can refer to each binary using '$(TOOL)' variable.
It will NOT install correct Version if missing.

%s

  list <flags> [<package or binary>]

List enumerates all or one binary that are/is currently pinned in this project. It will print exact path, Version and immutable output.

%s

  Version

Prints bingo Version.
`
