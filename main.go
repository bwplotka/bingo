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
}
