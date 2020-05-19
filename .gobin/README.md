# Project Development Dependencies.

This is directory which stores Go modules with pinned buildable package that is used within this repository, managed by https://github.com/bwplotka/gobin.

* Run `gobin get` to install all tools having each own module file in this directory.
* Run `gobin get <tool>` to install <tool> that have own module file in this directory.
* If `Makefile.binary-variables` is present, use $(<upper case tool name>) variable where <tool> is the <root>/.gobin/<tool>.mod.
* See https://github.com/bwplotka/gobin or -h on how to add, remove or change binaries dependencies.

## Requirements

* Go 1.14+
