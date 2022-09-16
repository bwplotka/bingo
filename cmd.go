// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/bwplotka/bingo/pkg/runner"

	"github.com/bwplotka/bingo/pkg/bingo"
	"github.com/bwplotka/bingo/pkg/version"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewBingoGetCommand(logger *log.Logger) *cobra.Command {
	var (
		goCmd    string
		rename   string
		name     string
		insecure bool
		link     bool
	)

	cmd := &cobra.Command{
		Use: "get [flags] [<package or binary>[@version1 or none,version2,version3...]]",
		Example: "bingo get github.com/fatih/faillint\n" +
			"bingo get github.com/fatih/faillint@latest\n" +
			"bingo get github.com/fatih/faillint@v1.5.0\n" +
			"bingo get github.com/fatih/faillint@v1.1.0,v1.5.0",
		Long: "go get like, simple CLI that allows automated versioning of Go package level \n" +
			"binaries(e.g required as dev tools by your project!) built on top of Go Modules, allowing reproducible dev environments.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if len(goCmd) == 0 {
				return errors.New("'go' flag cannot be empty")
			}
			if len(args) > 1 {
				return errors.New("too many arguments except none or binary/package")
			}
			if len(rename) > 0 && len(name) > 0 {
				return errors.New("Both -n and -r were specified. You can either rename or create new one.")
			}
			if len(name) > 0 && !regexp.MustCompile(`[a-zA-Z0-9.-_]+`).MatchString(name) {
				return errors.New("-n name contains not allowed characters")
			}
			if len(rename) > 0 && !regexp.MustCompile(`[a-zA-Z0-9.-_]+`).MatchString(rename) {
				return errors.New("-r name contains not allowed characters")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			r, err := runner.NewRunner(ctx, logger, insecure, goCmd)
			if err != nil {
				return err
			}
			if verbose {
				r.Verbose()
			}

			modDirAbs, err := filepath.Abs(moddir)
			if err != nil {
				return errors.Wrap(err, "abs")
			}
			defer func() {
				if err == nil {
					// Leave tmp files on error for debug purposes.
					if cerr := cleanGoGetTmpFiles(modDirAbs); cerr != nil {
						logger.Println("cannot clean tmp files", err)
					}
				}
			}()

			cfg := getConfig{
				runner:    r,
				modDir:    modDirAbs,
				relModDir: moddir,
				name:      name,
				rename:    rename,
				verbose:   verbose,
				link:      link,
			}
			var target string
			if len(args) > 0 {
				target = args[0]
			}
			if err := get(ctx, logger, cfg, target); err != nil {
				return errors.Wrap(err, "get")
			}

			pkgs, err := bingo.ListPinnedMainPackages(logger, modDirAbs, true)
			if err != nil {
				return errors.Wrap(err, "list pinned")
			}
			if len(pkgs) == 0 {
				return bingo.RemoveHelpers(modDirAbs)
			}
			return bingo.GenHelpers(moddir, version.Version, pkgs)
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&name, "name", "n", "", "The -n flag instructs to get binary and name it with given name instead of default,\n"+
		"so the last element of package directory. Allowed characters [A-z0-9._-]. If -n is used and no package/binary is specified,\n"+
		"bingo get will return error. If -n is used with existing binary name, copy of this binary will be done. Cannot be used with -r")
	flags.StringVarP(&rename, "rename", "r", "", "The -r flag instructs to get existing binary and rename it with given name. Allowed characters [A-z0-9._-]. \n"+
		"If -r is used and no package/binary is specified or non existing binary name is used, bingo will return error. Cannot be used with -n.")
	flags.StringVar(&goCmd, "go", "go", "Path to the go command.")
	flags.BoolVar(&insecure, "insecure", insecure, `Use -insecure flag when using 'go get'`)
	flags.BoolVarP(&link, "link", "l", link, "If enabled, bingo will also create soft link called <tool> that links to the current <tool>-<version> binary.\n"+
		"Use Variables.mk and variables.env if you want to be sure that what you are invoking is what is pinned.")
	return cmd
}

func NewBingoListCommand(logger *log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <flags> [<package or binary>]",
		Version: version.Version,
		Short:   "List enumerates all or one binary that are/is currently pinned in this project. ",
		Long:    "List enumerates all or one binary that are/is currently pinned in this project. It will print exact path, Version and immutable output.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return errors.New("too many arguments except none or binary/package")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			modDir, err := filepath.Abs(moddir)
			if err != nil {
				return errors.Wrap(err, "abs")
			}
			pkgs, err := bingo.ListPinnedMainPackages(logger, modDir, false)
			if err != nil {
				return err
			}
			var target string
			if len(args) > 0 {
				target = args[0]
			}
			bingo.SortRenderables(pkgs)
			return pkgs.PrintTab(target, os.Stdout)
		},
	}
	return cmd
}

func NewBingoVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Prints bingo Version.",
		Long:  `Prints bingo Version.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Bingo Version:", version.Version)
		},
	}
	return cmd
}
