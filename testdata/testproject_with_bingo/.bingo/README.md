# Development Dependencies.

This is directory which stores Go modules for each tools that is used within this repository, managed by https://github.com/bwplotka/bingo.

## Requirements

* Network (:
* Go 1.14+

## Usage

Just run `go get -modfile <root>/.bingo/<tool>.mod`to install tool in required version in your $(GOBIN).

### Within Makefile

Use $(<tool>) variable where <tool> is the <root>/.bingo/<tool>.mod.

This directory is managed by bingo tool.

* Run `go get -modfile <root>/.bingo/bingo.mod` if you did not before to install bingo.
* Run `bingo get` to install all tools in this directory.
* See https://github.com/bwplotka/bingo or -h on how to add, remove or change binaries dependencies.
