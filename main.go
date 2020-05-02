package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwplotka/gobin/pkg/gomodcmd"
	"github.com/oklog/run"
	"github.com/pkg/errors"
)

func main() {
	logger := log.New(os.Stderr, "gobin", log.Ldate|log.Ltime)
	flags := flag.NewFlagSet("gobin", flag.ExitOnError)
	modDir := flags.String("moddir", ".gobin", "Directory where separate module for tools is / will be"+
		" maintained. Any operation will be equivalent to go <cmd> -modfile <moddir>/go.mod.")
	goCmd := flags.String("go", "go", "Path to the go command.")

	if err := flags.Parse(os.Args[1:]); err != nil {
		logger.Fatalf("failed to parse flags: %v", err)
	}

	if *modDir == "" {
		logger.Fatal("'moddir' flag cannot be empty")
	}

	if *goCmd == "" {
		logger.Fatal("'go' flag cannot be empty")
	}

	if flags.NArg() == 0 {
		logger.Fatal("no command specified")
	}

	var cmdFunc func(ctx context.Context, r *gomodcmd.Runner) error
	switch flags.Arg(1) {
	case "get":
		gflags := flag.NewFlagSet("gobin get", flag.ExitOnError)
		insecure := gflags.Bool("insecure", false, "")
		update := gflags.Bool("u", false, "The -u flag instructs get to update modules providing dependencies of packages named on the command line to use newer minor or patch releases when available.")
		updatePatch := gflags.Bool("u=patch", false, "The -u=patch flag (not -u patch) also instructs get to update dependencies, but changes the default to select patch releases.")

		if err := gflags.Parse(os.Args[2:]); err != nil {
			logger.Fatalf("failed to parse flags for get command: %v", err)
		}
		upPolicy := noUpdatePolicy
		if *update {
			upPolicy = updatePolicy
		}
		if *updatePatch {
			upPolicy = updatePatchPolicy
		}
		cmdFunc = func(ctx context.Context, r *gomodcmd.Runner) error {
			return get(ctx, logger, r, *modDir, *insecure, upPolicy, gflags.Args()[1:]...)
		}
	case "list":
		// TODO.
	default:
		logger.Fatal("no such command", flags.Arg(1))
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
			r, err := gomodcmd.NewRunner(ctx, *modDir, *goCmd)
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
		logger.Fatalf("err: %+v", errors.Wrapf(err, "%s command failed", flags.Arg(1)))
	}
}
