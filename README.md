# gobin
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/bwplotka/gobin)
[![Latest Release](https://img.shields.io/github/release/bwplotka/gobin.svg?style=flat-square)](https://github.com/bwplotka/gobin/releases/latest)
[![CI](https://github.com/bwplotka/gobin/workflows/go/badge.svg)](https://github.com/bwplotka/gobin/actions?query=workflow%3Ago)
[![Go Report Card](https://goreportcard.com/badge/github.com/bwplotka/gobin)](https://goreportcard.com/report/github.com/bwplotka/gobin)

Simple CLI that automates versioning of Go binaries (e.g required as tools by your project!) in a nested Go module, allowing reproducible dev environment

## Goals

* Allow maintaining separate, clear go.mod for your Go tools without obfuscating your own (if you have any) one!
* Easy upgrade, downgrade, addition or removal of the tool version.
* Reliable way to make sure users and CIs are using exactly the same version of the tool.

## Requirements:

* Go 1.13+
* Linux or MacOS.
* Tools have to be build in Go and have to be [Go Modules] compatible.

## Usage / Example

Usage is simple, because `gobin` is just automating various existing `go` commands like `go mod init`, `go mod tidy`, `go get`
or `go install`.

The key idea is that **we want to maintain a separate, nested go module for our binaries. By default, it will be in `_gobin/go.mod`.**
This allows to solve our [goals](#Goals) without polluting main go module. Your project should commit all the files in this directory.

For example purposes, let's imagine our project requires a nice import formatting via external [`goimports`](https://pkg.go.dev/golang.org/x/tools/cmd/goimports?tab=doc)
binary (Actually it is recommend for all projects 🤓).

- [Adding a Go Tool / Starting](#adding-a-go-tool---starting)
- [Changing Version of a Tool & Installing](#changing-version-of-a-tool---installing)
- [Removing a Tool](#removing-a-tool)
- [Reliable Usage of a Tool](#reliable-usage-of-a-tool)

<small><i><a href='http://ecotrust-canada.github.io/markdown-toc/'>Table of contents generated with markdown-toc</a></i></small>

#### Adding a Go Tool / Starting

On repo without `gobin` used before, or with already existing `_gobin` directory, you can start by
adding a tool.

Similar to official way of adding dependencies, like `go get`, do:

`gobin get -u golang.org/x/tools/cmd/goimports`

If you don't pin the version it will use the latest available and pin that version in separate `_gobin/go.mod` module.

This will also **always** install the tool in a given version in you `${GOBIN}` path.

#### Changing Version of a Tool

If you want to update to the latest add `-u`, the same as `go get`:

`gobin get -u golang.org/x/tools/cmd/goimports`

If you want to pin to certain version, do as well same as `go get`:

`gobin get golang.org/x/tools/cmd/goimports@v0.0.0-20200502202811-ed308ab3e770`

Use `-o` option to change binary output name.

#### Removing a Tool

Exactly the same as native `go get`, just add `@none` and run `go get`:

`gobin get golang.org/x/tools/cmd/goimports@none`

#### Reliable Usage of a Tool

In you script or Makefile, try to always make sure the correct version of the tools are invoked.
Running just `goimports` is not enough, because user might have `goimports` in another path or installed different version
manually.

You can ensure correct binaries are used using simple trick.

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
format:
	@echo ">> formatting code"
	@go fmt -s -w $(FILES_TO_FMT)
	@$(GOIMPORTS) -w $(FILES_TO_FMT)
```

## Production Usage

To see production example see:

 * [gobin tools](https://github.com/bwplotka/gobin/blob/298d2bf5dcc1c8543261279f0a7a22536782e2b3/_gobin/binaries.go#L19)
 * [Thanos](WIP)
 * [go-grpc-middleware](WIP)

## Why your project need this?

* It's a key to pin version of the tools your project needs. Otherwise, users or CI will use different formatters, static analysis or build tools than the
given project version was build against and expected to run with. This can lead tons of support issues and confusion.
* There is currently no native and official way for pinning Go tools, especially without polluting your project's module.

## But hey, there is already some pattern for this!

Yes, but it's not perfect. We are referring to [this recommendation](https://github.com/golang/go/issues/25922#issuecomment-590529870).

There are a few downsides of the given recommendation (TL;DR: just `tools.go` inside your own module):

* Tools and your application share the same go modules. This can end up really badly because:

  1. It increases the risk of dependency hell: imagine you cannot update your app dependency because the tool depend on older version.
  2. Anyone who imports or installs your application *always* downloaded ALL tools and their dependencies. Do all users really need that `golangci-lint` in a certain version to
  just use your application?

* It does not guard you from accidental use of a tool you require (e.g static analysis or formatting) from a different version than your tooling expects.
It leads to confusing support tickets, confused users and just wasted time. Having reproducible tooling and development environment is crucial for reproducibility
and project maintainability, especially in CIs or different platforms.

* Manual addition the tool to "hacky" tools.go is prone to errors and just surprising for end contributors.

That's why we thought of building dedicated tool for this, allowing to use standard Go mechanisms like `modules`, `go run`, and `go get`.

In fact `gobin` is a little bit like extension of [this idea](https://github.com/golang/go/issues/25922#issuecomment-590529870).

## How this tool is different than [myitcv/gobin](https://github.com/myitcv/gobin)?

Looks like https://github.com/myitcv/gobin was created mainly to tackle running reproducibility with wrapping `go run`.

This `gobin` have a bit wider [Goals](#Goals).

## TODO:

* [ ] e2e tests.
* [ ] go run via gobin?
