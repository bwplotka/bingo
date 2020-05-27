# bingo
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/bwplotka/bingo)
[![Latest Release](https://img.shields.io/github/release/bwplotka/bingo.svg?style=flat-square)](https://github.com/bwplotka/bingo/releases/latest)
[![CI](https://github.com/bwplotka/bingo/workflows/go/badge.svg)](https://github.com/bwplotka/bingo/actions?query=workflow%3Ago)
[![Go Report Card](https://goreportcard.com/badge/github.com/bwplotka/bingo)](https://goreportcard.com/report/github.com/bwplotka/bingo)

Simple CLI that allows automated versioning of Go package level binaries (e.g required as dev tools by your project!) built on top of `go` command modules, allowing reproducible dev environments.

## Problem Statement

From our experience all repositories and projects require some tools and binaries to be present on the machine to be able to perform various development
operations like building, formatting, releasing or static analysis. For smooth development all such tools should be pinned to a certain version and
bounded to the code commits there were meant to be used against.

Go modules were not aimed to solve this problem, and even if they will do at some point it will not be on the package level, which makes it impossible to e.g
pin minor version `X.Y.0` of package `module1/cmd/abc` and version `X.Z.0` of `module1/cmd/def`.

The alternatives are problematic as well:

* Hosting of downloading prebuild binaries is painful due to variety of CPU architectures and OS distributions and versions.
* Managing this via package managers is painful as different OS-es has different package managers (`dep`, `yum`, `brew`, `snap` or thousands of others),
 plus most of the tools are either super old or not present in those.
* Fancy: Docker image build with all tools needed. This is kind of great, but comes with tradeoff of long bake times, tool run times and sharing files
between docker guest and host is always problematic (permissions, paths etc).

While maintaining larger projects like [Thanos](http://thanos.io/), [Prometheus](http://prometheus.io), [grpc middlewares](https://github.com/grpc-ecosystem/go-grpc-middleware), we
 found that:

* Most of the tools we require was written in Go, usually as sub package (not necessarily a module!). It's very easy to write, efficient robust and cross-platform tools in this language.
* Go versioning is not suited for building such tools, however with some clean strategy and couple of Go commands you can version required tools.

This is how `bingo` tool was born. Just run `bingo get <github.com/toolorg/tool/cmd/tool@versionIWant>` to start! Also make sure
to use `-m` option if your projects uses `Makefile` to generate useful variables.

Read full story about this tool [here](WIP).

  * [Problem Statement](#problem-statement)
  * [Goals](#goals)
  * [Requirements](#requirements)
  * [Contributing](#contributing)
  * [Usage](#usage)
    + [Installing bingo](#installing-bingo)
    + [Adding a Go Tool](#adding-a-go-tool)
    + [Changing Version of a Tool](#changing-version-of-a-tool)
    + [Removing a Tool](#removing-a-tool)
    + [Reliable Usage of a Tool](#reliable-usage-of-a-tool)
    + [Getting correct binary using go (+ optionally Makefile!)](#getting-correct-binary-using-go--optionally-makefile)
    + [Getting correct binary using bingo](#getting-correct-binary-using-bingo)
  * [Production Usage](#production-usage)
  * [Why your project need this?](#why-your-project-need-this)
  * [Why not a simple `_tools` go pattern?](#why-not-a-simple-_tools-go-pattern)
  * [How this tool is different to [myitcv/gobin](https://github.com/myitcv/gobin)?](#how-this-tool-is-different-to-myitcvgobin)
  * [Initial Author](#initial-author)

<small><i><a href='http://ecotrust-canada.github.io/markdown-toc/'>Table of contents generated with markdown-toc</a></i></small>


<small><i><a href='http://ecotrust-canada.github.io/markdown-toc/'>Table of contents generated with markdown-toc</a></i></small>

## Goals

* Allow maintaining separate, hidden, nested Go modules for Go buildable packages you need **without obfuscating your own module**!
    * Also works for non-Go projects, requiring tools that just happen to be written in Go (:
    * Option to NOT require `bingo` for installation of tools.
* Easy upgrade, downgrade, addition or removal of the needed binary's version, with no risk of dependency conflicts.
    * NOTE: Tools are **often** not following semantic versioning, so they need to be pinned by commit.
* Reliable way to make sure users and CIs are using expected version of the binaries, with reinstall on demand only if needed.
* Package level versioning, which allows versioning different packages from single module in different versions.
* Bulk versioning: pinning of multiple versions of binaries from the same Go package (e.g for compatibility tests).
* Optional, easy integration with Makefiles.

## Requirements

* Go 1.14+
* Linux or MacOS.
* Tools have to be build in Go and have to be [Go Modules] compatible.

## Contributing

Any contributions are welcome! Just use GitHub Issues and Pull Requests as usual. We follow [Thanos Go coding style](https://thanos.io/contributing/coding-style-guide.md/) guide.

## Usage

Usage is simple, because `bingo` is just automating various existing `go` commands like `go mod init`, `go build`, `go get` etc.

The key idea is that **we want to maintain a separate, nested go module for each of the binary. By default, it will be in `.bingo/<tool>.mod`.**
This allows to solve our [goals](#Goals) without polluting main go module. Your project should commit all not gitignored files from `.bingo` directory.

For example purposes, let's imagine our project requires a nice import formatting via external [`goimports`](https://pkg.go.dev/golang.org/x/tools/cmd/goimports?tab=doc)
binary (Actually it is recommend for all projects 🤓).

### Installing bingo

`go get github.com/bwplotka/bingo`

### Adding a Go Tool

On repo without `bingo` used before, or with already existing `.bingo` directory, you can start by
adding a binary (tool). Similar to official way of adding dependencies, like `go get`, do:

`bingo get golang.org/x/tools/cmd/goimports`

If you don't pin the version it will use the latest available and pin that version in separate `.bingo/goimports.mod` module.

This will also install the **immutable tool** under `${GOBIN}/goimports-<version>`. Binaries are versions to make sure
proper one is used when needed, which does not require tooling to verify checksums etc.

### Changing Version of a Tool

If you want to update to the latest add `-u`, the same as `go get`:

`bingo get -u golang.org/x/tools/cmd/goimports`

If you want to pin to certain version, do as well same as `go get`:

`bingo get golang.org/x/tools/cmd/goimports@v0.0.0-20200502202811-ed308ab3e770`

This will produce `${GOBIN}/goimports-v0.0.0-20200502202811-ed308ab3e770`

Use `-n` option to change binary base (reference) name. For example:

`bingo get -n goimports2 golang.org/x/tools/cmd/goimports@v0.0.0-20200502202811-ed308ab3e770`

Will produce `${GOBIN}/goimports2-v0.0.0-20200502202811-ed308ab3e770`

### Removing a Tool

Exactly the same as native `go get`, just add `@none` and run `go get`:

`bingo get golang.org/x/tools/cmd/goimports@none` or simply `bingo get goimports@none` if `-n` was `goimports` or `-n` was not specified,
when you were installing the tool the first time.

### Reliable Usage of a Tool

`gobin get` automatically installs tools in pinned version into the ${GOBIN}.

Thanks to immutable output names, which is done by adding version suffix, we should be certain that we use correct version of the tool.
Otherwise, for example, running "just" `goimports` can be problematic, because user might have `goimports` in another path or might
have installed a different version.

However, obviously, those binaries should NOT be checked into your VCS. Let's go now through ways you can ensure
users of your projects, as well as CI can install pinned tools.

You can ensure correct version of the binaries are used using following patterns:

### Getting correct binary using go (+ optionally Makefile!)

From project's root, run:

`go build -modfile .bingo/<tool>.mod -o=<where you want to build> <tool package>`

This is quite amazing, as you can use / install those binaries without `bingo`. This makes `bingo` only necessary if
you want to update / downgrade / remove or add the tool.

However such build command is bit complex. That's why you can get advantage from `Makefile`!

When using `bingo get`, add `-m`. This will create `.bingo/Variables.mk` and attempt to include this in your `Makefile`.

Thanks to that you can refer to the binary using `$(TOOL)` variable which will install correct version if missing.

### Getting correct binary using bingo

`bingo get goimports` or just `bingo get` to install all tools specified in `.bingo` dir.

## Production Usage

To see production example see:

 * [bingo's own tools](https://github.com/bwplotka/bingo/tree/master/.bingo)
 * [Thanos](WIP)
 * [go-grpc-middleware](WIP)

## Why your project need this?

* It's a key to pin version of the tools your project needs. Otherwise, users or CI will use different formatters, static analysis or build tools than the
given project version was build against and expected to run with. This can lead tons of support issues and confusion.

* There is currently no native and official way for pinning Go tools, especially without polluting your project's module.

## Why not a simple `_tools`.go pattern

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

That's why we thought of building dedicated tool for this, allowing to use standard Go mechanisms like `go build` and `go get`.

* tools pattern does not allow to pin multiple different versions of the same package (or multiple version of differnet packages within the same module).

In fact `bingo` is a little bit like extension of [this idea](https://github.com/golang/go/issues/25922#issuecomment-590529870).

## How this tool is different to [myitcv/gobin](https://github.com/myitcv/gobin)?

Looks like https://github.com/myitcv/gobin was created mainly to tackle running reproducibility with wrapping `go run`. Whereas
`bingo` have a bit wider [Goals](#Goals).

## Initial Author

[@bwplotka](https://bwplotka.dev)
