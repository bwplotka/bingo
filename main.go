// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/bwplotka/gobin/pkg/gobin"
	"github.com/bwplotka/gobin/pkg/gomodcmd"
	"github.com/oklog/run"
	"github.com/pkg/errors"
)

const (
	defaultMakefileName = "Makefile"
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
	flags := flag.NewFlagSet("gobin", flag.ExitOnError)
	verbose := flags.Bool("v", false, "Print more'")

	// Get flags.
	getFlags := flag.NewFlagSet("gobin get", flag.ExitOnError)
	getModDir := getFlags.String("moddir", ".gobin", "Directory where separate modules for each binary will be "+
		"maintained. Feel free to commit this directory to your VCS to bond binary versions to your project code. If relative"+
		"path is used, it is expected to be relative to project root module.")
	output := getFlags.String("o", "", "The -o flag instructs to build with certain output name. Allowed characters [A-z0-9._-]. "+
		"The given output will be then used to reference a binary later on. If empty the last element of package directory will be used "+
		"If no package/binary is specified, gobin get will return error")
	goCmd := getFlags.String("go", "go", "Path to the go command.")
	update := getFlags.Bool("u", false, "The -u flag instructs get to update modules providing dependencies of packages named on the command line to use newer minor or patch releases when available.")
	updatePatch := getFlags.Bool("upatch", false, "The -upatch flag (not -u patch) also instructs get to update dependencies, but changes the default to select patch releases.")
	insecure := getFlags.Bool("insecure", false, "Use -insecure flag when using 'go get'")
	makefile := getFlags.String("makefile", defaultMakefileName, "Makefile to link the the generated helper for make when `-m` options is specified with."+
		"Specify empty to disable including the helper.")
	genMakefileHelper := getFlags.Bool("m", false, "Generate makefile helper with all binaries as variables.")
	// Go flags is so broken, need to add shadow -v flag to make those work in both before and after `get` command.
	getVerbose := getFlags.Bool("v", false, "Print more'")

	// List flags.
	listFlags := flag.NewFlagSet("gobin list", flag.ExitOnError)
	listModDir := listFlags.String("moddir", ".gobin", "Directory where separate modules for each binary will be "+
		"maintained. Feel free to commit this directory to your VCS to bond binary versions to your project code. If relative"+
		"path is used, it is expected to be relative to project root module.")
	// Go flags is so broken, need to add shadow -v flag to make those work in both before and after `list` command.
	listVerbose := listFlags.Bool("v", false, "Print more'")

	flags.Usage = func() {
		getFlagsHelp := &strings.Builder{}
		getFlags.SetOutput(getFlagsHelp)
		getFlags.PrintDefaults()

		listFlagsHelp := &strings.Builder{}
		listFlags.SetOutput(listFlagsHelp)
		listFlags.PrintDefaults()
		fmt.Printf(`gobin: Simple CLI that automates versioning of Go binaries (e.g required as tools by your project!) in a nested Go module, allowing reproducible dev environments.

For detailed examples see: https://github.com/bwplotka/gobin

Commands:

	get <flags> [<package or binary>[@version or none]]

%s

	list <flags> [<package or binary>]

%s

	version

Prints gobin version.

`, getFlagsHelp.String(), listFlagsHelp.String())
	}
	if err := flags.Parse(os.Args[1:]); err != nil {
		exitOnUsageError(flags.Usage, "Failed to parse flags:", err)
	}
	if flags.NArg() == 0 {
		exitOnUsageError(flags.Usage, "No command specified")
	}
	var cmdFunc func(ctx context.Context, r *gomodcmd.Runner) error
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

		upPolicy := gomodcmd.NoUpdatePolicy
		if *update {
			upPolicy = gomodcmd.UpdatePolicy
		}
		if *updatePatch {
			upPolicy = gomodcmd.UpdatePatchPolicy
		}

		if getFlags.NArg() > 1 {
			exitOnUsageError(flags.Usage, "Too many arguments except none or binary/package ")
		}

		target := getFlags.Arg(0)
		if *output != "" && target == "" {
			exitOnUsageError(flags.Usage, "Cannot set output -o with no package/binary specified")
		}

		if *output != "" && !regexp.MustCompile(`[a-zA-Z0-9.-_]+`).MatchString(*output) {
			exitOnUsageError(flags.Usage, *output, " present as -o contains not allowed characters")
		}

		cmdFunc = func(ctx context.Context, r *gomodcmd.Runner) error {
			rootDir, err := r.With(ctx, "", "").List("-m", "-f={{ .Dir }}")
			if err != nil {
				return err
			}

			modDir := *getModDir
			if !filepath.IsAbs(modDir) {
				modDir = filepath.Join(rootDir, modDir)
			}

			// Like go get, but package aware and without go source files.
			if err := get(ctx, r, modDir, upPolicy, target, *output); err != nil {
				return err
			}

			// Remove all sum files. This is garbage for us.
			if err := removeAllGlob(filepath.Join(modDir, "*.sum")); err != nil {
				return err
			}

			modFiles, err := filepath.Glob(filepath.Join(modDir, "*.mod"))
			if err != nil {
				return err
			}
			if len(modFiles) == 0 {
				return gobin.RemoveMakeHelper(modDir)
			}

			// Get through all modules and remove those without gobin metadata. This ensures we clean
			// non-gobin maintained module files from this directory as well partial module files.
			for _, f := range modFiles {
				has, err := gobin.ModHasMeta(f, nil)
				if err != nil {
					return err
				}
				if !has {
					logger.Println("found malformed module file, removing:", f)
					if err := os.RemoveAll(strings.TrimSuffix(f, ".") + "*"); err != nil {
						return err
					}
				}
			}

			if !*genMakefileHelper {
				return nil
			}

			if *makefile != "" {
				if filepath.IsAbs(*makefile) {
					return errors.Errorf("makefile has to be a relative path, got: %v", *makefile)
				}
				*makefile = filepath.Join(rootDir, *makefile)
			}

			// Create makefile helper.
			return gobin.GenMakeHelperAndHook(modDir, *makefile, version, modFiles...)
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
		cmdFunc = func(ctx context.Context, r *gomodcmd.Runner) error {
			rootDir, err := r.With(ctx, "", "").List("-m", "-f={{ .Dir }}")
			if err != nil {
				return err
			}

			modDir := *listModDir
			if !filepath.IsAbs(modDir) {
				modDir = filepath.Join(rootDir, modDir)
			}

			modFiles, err := filepath.Glob(filepath.Join(modDir, "*.mod"))
			if err != nil {
				return err
			}

			var targets []string
			for _, f := range modFiles {
				has, err := gobin.ModHasMeta(f, nil)
				if err != nil {
					return err
				}
				if !has {
					continue
				}
				if target != "" {
					// TODO(bwplotka): Allow per packages?
					if target+".mod" == filepath.Base(f) {
						targets = append(targets, f)
						break
					}
					continue
				}
				targets = append(targets, f)
			}

			if len(targets) == 0 {
				if target == "" {
					return nil
				}
				exitOnUsageError(flags.Usage, "No binaries found for", target)
			}
			return list(targets)
		}
	case "version":
		cmdFunc = func(ctx context.Context, r *gomodcmd.Runner) error {
			_, err := fmt.Fprintln(os.Stdout, version)
			return err
		}
	default:
		exitOnUsageError(flags.Usage, "No such command", flags.Arg(0))
	}

	g := &run.Group{}
	// Listen for signal interrupts.
	{
		cancel := make(chan struct{})
		g.Add(func() error {
			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
			select {
			case s := <-c:
				return errors.Errorf("caught signal %q; exiting. ", s)
			case <-cancel:
				return nil
			}
		}, func(error) {
			close(cancel)
		})
	}

	// Run command.
	{
		ctx, cancel := context.WithCancel(context.Background())
		g.Add(func() error {
			r, err := gomodcmd.NewRunner(ctx, *insecure, *goCmd)
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

func list(modFiles []string) error {
	for _, f := range modFiles {
		pkg, ver, err := gobin.ModDirectPackage(f, nil)
		if err != nil {
			return errors.Wrapf(err, "module %q is malformed. 'get' full package name to re-pin it.", f)
		}
		fmt.Printf("%s: %s@%s\n", filepath.Base(strings.TrimSuffix(f, ".mod")), pkg, ver)
	}
	return nil
}
