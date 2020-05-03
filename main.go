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
	"strings"
	"syscall"

	"github.com/bwplotka/gobin/pkg/gomodcmd"
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

	flags := flag.NewFlagSet("gobin", flag.ExitOnError)
	modDir := flags.String("moddir", "_gobin", "Directory where separate module for tools is / will be"+
		" maintained. Any operation will be equivalent to go <cmd> -modfile <moddir>/go.mod.")
	goCmd := flags.String("go", "go", "Path to the go command.")
	insecure := flags.Bool("insecure", false, "")
	usage := &strings.Builder{}
	flags.SetOutput(usage)

	// Get flags.
	getFlags := flag.NewFlagSet("gobin get", flag.ExitOnError)
	update := getFlags.Bool("u", false, "The -u flag instructs get to update modules providing dependencies of packages named on the command line to use newer minor or patch releases when available.")
	updatePatch := getFlags.Bool("u=patch", false, "The -u=patch flag (not -u patch) also instructs get to update dependencies, but changes the default to select patch releases.")
	output := getFlags.String("o", "", "The -o flag instructs to build with certain output name. If none or more than one package is specified, get will return error")
	getFlags.SetOutput(usage)

	flags.Usage = func() {
		flags.PrintDefaults()
		getFlags.PrintDefaults()
		fmt.Printf(`gobin: is a CLI for a clean and reproducible module-based management of all Go binaries your project requires for the development.

For detailed examples see: https://github.com/bwplotka/gobin

Command:

gobin get <flags> [packages@version] [packages1@version ]

%s
`, usage.String())
	}

	if err := flags.Parse(os.Args[1:]); err != nil {
		exitOnUsageError(flags.Usage, "Failed to parse flags:", err)
	}

	if *modDir == "" {
		exitOnUsageError(flags.Usage, "'moddir' flag cannot be empty")
	}

	if *goCmd == "" {
		exitOnUsageError(flags.Usage, "'go' flag cannot be empty")
	}

	if flags.NArg() == 0 {
		exitOnUsageError(flags.Usage, "No command specified")
	}

	var cmdFunc func(ctx context.Context, r *gomodcmd.Runner) error
	switch flags.Arg(0) {
	case "get":
		if err := getFlags.Parse(os.Args[2:]); err != nil {
			exitOnUsageError(flags.Usage, "Failed to parse flags for get command:", err)
		}
		upPolicy := gomodcmd.NoUpdatePolicy
		if *update {
			upPolicy = gomodcmd.UpdatePatchPolicy
		}
		if *updatePatch {
			upPolicy = gomodcmd.UpdatePatchPolicy
		}

		packages := getFlags.Args()
		if *output != "" && len(packages) != 1 {
			exitOnUsageError(flags.Usage, "Cannot set output -o with %v packages", packages)
		}

		cmdFunc = func(ctx context.Context, r *gomodcmd.Runner) error {
			return get(ctx, logger, r, filepath.Join(*modDir, "binaries.go"), upPolicy, packages...)
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
			r, err := gomodcmd.NewRunner(ctx, *insecure, *modDir, *goCmd)
			if err != nil {
				return err
			}
			return cmdFunc(ctx, r)
		}, func(error) {
			cancel()
		})
	}
	if err := g.Run(); err != nil {
		// Use %+v for github.com/pkg/errors error to print with stack.
		logger.Fatalf("Error: %+v", errors.Wrapf(err, "%s command failed", flags.Arg(1)))
	}
}
