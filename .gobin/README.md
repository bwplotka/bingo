# Development Dependencies.

This is directory which stores Go modules for each tools that is used within this repository, managed by https://github.com/bwplotka/gobin.

## Requirements

* Network (:
* Go 1.14+

## Usage

Just run `go get -modfile /home/bwplotka/Repos/gobin/.gobin/<tool>.mod`to install tool in required version in your $(GOBIN).

### Within Makefile

Use $(<tool>) variable where <tool> is the /home/bwplotka/Repos/gobin/.gobin/<tool>.mod.

This directory is managed by gobin tool.

* Run `go get -modfile /home/bwplotka/Repos/gobin/.gobin/gobin.mod` if you did not before to install gobin.
* Run `gobin get` to install all tools in this directory.
* See https://github.com/bwplotka/gobin or -h on how to add, remove or change binaries dependencies.
