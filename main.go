// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

//lint:file-ignore faillint main.go can use fmt.Print* family for error logging, when logger is not ready.

package main

import (
	"log"
	"os"

	"github.com/spf13/cobra"
)

var verbose bool
var moddir string

func NewBingoCommand(logger *log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bingo",
		Short: "Bingo is a command line tool for managing your Bing-o installation.",
		Long: "Bingo is a command line tool for Like `go get`  but for Go tools! \n" +
			"CI Automating versioning of Go binaries in a nested, isolated Go modules.\n" +
			"For detailed examples and documentation see: https://github.com/bwplotka/bingo",
	}
	flags := cmd.PersistentFlags()
	flags.BoolVarP(&verbose, "verbose", "v", false, "Print more")
	flags.StringVarP(&moddir, "moddir", "m", ".bingo", "Directory where separate modules for each binary will be maintained. \n"+
		"Feel free to commit this directory to your VCS to bond binary versions to your project code. \n"+
		"If the directory does not exist bingo logs and assumes a fresh project.")
	cmd.AddCommand(NewBingoGetCommand(logger))
	cmd.AddCommand(NewBingoListCommand(logger))
	cmd.AddCommand(NewBingoVersionCommand())
	return cmd
}

func main() {
	logger := log.New(os.Stderr, "", 0)
	rootCmd := NewBingoCommand(logger)
	err := rootCmd.Execute()
	if err != nil {
		logger.Println(err)
		os.Exit(1)
	}
	return
}

const bingoHelpFmt = `bingo: 'go get' like, simple CLI that allows automated versioning of Go package level binaries (e.g required as dev tools by your project!)
built on top of Go Modules, allowing reproducible dev environments. 'bingo' allows to easily maintain a separate, nested Go Module for each binary.

For detailed examples and documentation see: https://github.com/bwplotka/bingo

'bingo' supports following commands:

Commands:

  get <flags> [<package or binary>[@version1,none,latest,version2,version3...]]

%s

  list <flags> [<package or binary>]

List enumerates all or one binary that are/is currently pinned in this project. It will print exact path, Version and immutable output.

%s

  version

Prints bingo Version.
`
