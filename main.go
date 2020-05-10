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
	version             = "v0.9.0"
	defaultMakefileName = "Makefile"
	gobinBinName        = "gobin"
	gobinInstallPath    = "github.com/bwplotka/gobin"
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
	modDir := getFlags.String("moddir", ".gobin", "Directory where separate modules for each binary will be "+
		"maintained. Feel free to commit this directory to your VCS to bond binary versions to your project code. If relative"+
		"path is used, it is expected to be relative to project root module.")
	output := getFlags.String("o", "", "The -o flag instructs to build with certain output name. Allowed characters [A-z0-9._-]. "+
		"The given output will be then used to reference a binary later on. If empty the last element of package directory will be used "+
		"If no package/binary is specified, gobin get will return error")
	goCmd := getFlags.String("go", "go", "Path to the go command.")
	update := getFlags.Bool("u", false, "The -u flag instructs get to update modules providing dependencies of packages named on the command line to use newer minor or patch releases when available.")
	updatePatch := getFlags.Bool("u=patch", false, "The -u=patch flag (not -u patch) also instructs get to update dependencies, but changes the default to select patch releases.")
	insecure := getFlags.Bool("insecure", false, "Use -insecure flag when using 'go get'")
	makefile := getFlags.String("makefile", "", "Makefile to use for generated helper for make. It is expected to be relative to project root module. If not found, no helper will be generated. If flag is specified but file was not found, it will return error")
	// Go flags are so broken, need to add shadow -v flag to make those work in both before and after `get` command.
	getVerbose := getFlags.Bool("v", false, "Print more'")

	flags.Usage = func() {
		getFlagsHelp := &strings.Builder{}
		getFlags.SetOutput(getFlagsHelp)
		getFlags.PrintDefaults()
		fmt.Printf(`gobin: Simple CLI that automates versioning of Go binaries (e.g required as tools by your project!) in a nested Go module, allowing reproducible dev environments.

For detailed examples see: https://github.com/bwplotka/gobin

Commands:

	get <flags> [<package or binary>[@version or none]]

%s

	version

Prints gobin version.

`, getFlagsHelp.String())
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
			fmt.Println("err")
			exitOnUsageError(flags.Usage, "Failed to parse flags for get command:", err)
		}

		if !*verbose && *getVerbose {
			*verbose = true
		}

		if *modDir == "" {
			exitOnUsageError(flags.Usage, "'moddir' flag cannot be empty")
		}

		if *goCmd == "" {
			exitOnUsageError(flags.Usage, "'go' flag cannot be empty")
		}

		upPolicy := gomodcmd.NoUpdatePolicy
		if *update {
			upPolicy = gomodcmd.UpdatePatchPolicy
		}
		if *updatePatch {
			upPolicy = gomodcmd.UpdatePatchPolicy
		}

		if getFlags.NArg() > 1 {
			exitOnUsageError(flags.Usage, "Too many arguments except none or binary/package ")
		}

		target := getFlags.Arg(0)
		if *output != "" && target != "" {
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

			modDir := *modDir
			if !filepath.IsAbs(modDir) {
				modDir = filepath.Join(rootDir, modDir)
			}

			if err := ensureGobinModFile(ctx, r, modDir); err != nil {
				return err
			}

			// Like go get, but package aware!
			if err := get(ctx, r, modDir, upPolicy, target, *output); err != nil {
				return err
			}

			modFiles, err := filepath.Glob(filepath.Join(modDir, "*.mod"))
			if err != nil {
				return err
			}

			// Get through all modules and remove those without meta.
			for _, f := range modFiles {
				has, err := gobin.ModHasMeta(f, nil)
				if err != nil {
					return err
				}
				if !has {
					if err := os.RemoveAll(f); err != nil {
						return err
					}
				}
			}

			// Generate makefile is Makefile exists.
			makeFile := *makefile
			if makeFile == "" {
				makeFile = defaultMakefileName
			}
			if !filepath.IsAbs(makeFile) {
				makeFile = filepath.Join(rootDir, makeFile)
			}

			if *makefile == "" {
				if _, err := os.Stat(makeFile); err != nil {
					// Makefile was not specified, so we do best effort makefile lookup. If it's not found in default location,
					// don't generate any helper.
					return nil
				}
			}

			// Create makefile helper.
			// TODO(bwplotka): Allow different tool name.
			return gobin.GenMakeHelperAndHook(makeFile, version, gobinInstallPath, gobinBinName, modFiles...)
		}
	case "version":
		cmdFunc = func(ctx context.Context, r *gomodcmd.Runner) error {
			_, err := fmt.Fprintln(os.Stdout, version)
			return err
		}
	default:
		exitOnUsageError(flags.Usage, "No such command", flags.Arg(1))
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
