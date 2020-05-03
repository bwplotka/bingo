# gobin
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/bwplotka/flagarize)
[![Latest Release](https://img.shields.io/github/release/bwplotka/flagarize.svg?style=flat-square)](https://github.com/bwplotka/flagarize/releases/latest)
[![CI](https://github.com/bwplotka/flagarize/workflows/test/badge.svg)](https://github.com/bwplotka/flagarize/actions?query=workflow%3Atest)
[![Go Report Card](https://goreportcard.com/badge/github.com/bwplotka/flagarize)](https://goreportcard.com/report/github.com/bwplotka/flagarize)

Tiny `go` command for a clean and reproducible module-based management of all Go binaries your project requires for the development.

## Goals

* Allow maintaining separate, clear go.mod for your Go tools without obfuscating your own (if you have any) one!
* Easy upgrade, downgrade, addition or removal of the tool version.
* Reliable way to make sure users and CIs are using exactly the same version of the tool.

## Requirements:

* Go 1.14+
* Linux or MacOS.
* Tools have to be build in Go and have to be [Go Modules] compatible.

## Usage / Example

The key idea is that we want to maintain separate go module for our binaries. By default it will be in `.gobin/go.mod`.
Your project should commit all the files in this directory.

Let's imagine our project requires nice import formatting via external [`goimports`](https://pkg.go.dev/golang.org/x/tools/cmd/goimports?tab=doc) binary (It should!).

### Adding a Go Tool (even for empty repo)

Similar to official way, just go get!

`gobin get -u golang.org/x/tools/cmd/goimports`

If you don't pin the version it will use the latest.

### Changing version of a Tool & Installing

If you want to update to the latest:

`gobin get -u golang.org/x/tools/cmd/goimports`

If you want to pin to certain version:

`gobin get golang.org/x/tools/cmd/goimports@v0.0.0-20200502202811-ed308ab3e770`

Use `-o` option to change binary output name.

### Removing a Tool

Exactly the same as native `go get`:

`gobin get golang.org/x/tools/cmd/goimports@none`

### Reliable Usage of a Tool

In you script or Makefile, try to always make sure correct version of the tools are invoked.
You can ensure correct binaries are used using following ways:

#### go run

Just always use tools using either:

* Go 1.14+:

`go run -modfile=_gobin/go.mod golang.org/x/tools/cmd/goimports`

* Go older than 1.14:

`cd _gobin/go.mod && go run golang.org/x/tools/cmd/goimports`

Don't worry about compiling it all the time. Thanks to amazing Go Team, all is cached ❤️

Not if you use Makefile it is as easy as:

```Makefile
GOIMPORTS ?= go run -modfile=_gobin/go.mod golang.org/x/tools/cmd/goimports

.PHONY: format
format: ## Formats Go code including imports.
format: $(GOIMPORTS)
	@echo ">> formatting code"
	@go fmt -s -w $(FILES_TO_FMT)
	@$(GOIMPORTS) -w $(FILES_TO_FMT)
```

## Production Usage

To see production example see:

 * [gobin tools](WIP)
 * [Thanos](WIP)
 * [go-grpc-middleware](WIP)

## Why my project need this?

* It's a key to pin version of the tools your project needs. Otherwise users or CI will use different formatters, static analysis or build tools than the
given project version was build against.
* There is currently no native and official for pinning Go tools, especially without malforming your project's module.

## But hey, there is already some pattern for this!

Yes, but it's not perfect. This wrapper is actually an extension of [this recommendation](https://github.com/golang/go/issues/25922#issuecomment-590529870).

There are few downsides of the old recommendation way (just tools.go inside your own module):

* Tools and your application share the same go modules. This can be really bad because:

  1. It increases the risk of dependency hell: imagine you cannot update your app dependency because the tool depend on older version.
  2. Anyone who imports or installs your application *always* downloaded ALL tools and their dependencies. Do all users really need that `golangci-lint` in a certain version to
  just use your application?

* It does not guard you from accidental use of a tool you require (e.g static analysis or formatting) from a different version than your tooling expects.
It leads to confusing support tickets, confused users and just wasted time. Having reproducible tooling and development environment is crucial for reproducibility
and project maintainability, especially in CIs or different platforms.

* Manual addition the tool to "hacky" tools.go is prone to errors and just surprising for end contributors.

That's why we thought of building dedicated tool for this, allowing to use standard Go mechanisms like `modules`, `go run`, and `go get`.

## How this tool is different than [myitcv/gobin](https://github.com/myitcv/gobin)?

Looks like https://github.com/myitcv/gobin was created mainly to tackle running reproducibility with wrapping `go run`.

This `gobin` have a bit wider [Goals](#Goals)

## TODO:

* [ ] e2e tests.
* [ ] List command?
* [ ] go run via gobin?