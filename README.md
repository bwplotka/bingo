# bingo

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/bwplotka/bingo) [![Latest Release](https://img.shields.io/github/release/bwplotka/bingo.svg?style=flat-square)](https://github.com/bwplotka/bingo/releases/latest) [![CI](https://github.com/bwplotka/bingo/workflows/go/badge.svg)](https://github.com/bwplotka/bingo/actions?query=workflow%3Ago) [![Go Report Card](https://goreportcard.com/badge/github.com/bwplotka/bingo)](https://goreportcard.com/report/github.com/bwplotka/bingo) [![Slack](https://img.shields.io/badge/join%20slack-%23bingo-brightgreen.svg)](https://gophers.slack.com/)

`go get` like, simple CLI that allows automated versioning of Go package level binaries (e.g required as dev tools by your project!) built on top of Go Modules, allowing reproducible dev environments.

[![Demo](examples/bingo-demo.gif)](examples/)

## Features

From our experience all repositories and projects require some tools and binaries to be present on the machine to be able to perform various development operations like building, formatting, releasing or static analysis. For smooth development all such tools should be pinned to a certain version and bounded to the code commits there were meant to be used against.

Go modules does not aim to solve this problem, and even if they will do at some point it will not be on the package level, which makes it impossible to e.g pin minor version `X.Y.0` of package `module1/cmd/abc` and version `X.Z.0` of `module1/cmd/def`.

At the end `bingo`, has following features:

* It allows maintaining separate, hidden, nested Go modules for Go buildable packages you need **without obfuscating your own module or worrying with tool's cross dependencies**!
* Package level versioning, which allows versioning different (or the same!) package multiple times from a single module in different versions.
* Works also for non-Go projects. It only requires the tools to be written in Go.
* No need to install `bingo` in order to **use** pinned tools. This avoids the "chicken & egg" problem. You only need `go build`.
* Easy upgrade, downgrade, addition, and removal of the needed binary's version, with no risk of dependency conflicts.
  * NOTE: Tools are **often** not following semantic versioning, so `bingo` allows to pin by commit ID.
* Immutable binary names. This creates a reliable way for users and CIs to use expected version of the binaries, reinstalling on-demand only if needed.
* Works with all buildable Go projects, including pre Go modules and complex projects with complex directives like `replace`, `retract` or `exclude` statements. (e.g Prometheus)
* Optional, automatic integration with Makefiles.

You can read full a story behind `bingo` [in this blog post](https://www.bwplotka.dev/2020/bingo/).

## Requirements

* Go 1.17+
* Linux or MacOS (Want Windows support? [Helps us out](https://github.com/bwplotka/bingo/issues/26))
* All tools that you wish to "pin" have to be built in Go (they don't need to use Go modules at all).

## Installing

In your repository (does not need to be a Go project)

```shell
go install github.com/bwplotka/bingo@latest
```

> For [go version before 1.17](https://go.dev/doc/go-get-install-deprecation) use `go get github.com/bwplotka/bingo` instead.

Recommended: Ideally you want to pin `bingo` tool to the single version too (inception!). Do it via:

```shell
bingo get -l github.com/bwplotka/bingo
```

## Usage

### `go get` but for binaries!

The key idea is that you can manage your tools similar to your Go dependencies via `go get`:

```shell
bingo get [<package or binary>[@version1 or none,version2,version3...]]
```

For example:

* `bingo get github.com/fatih/faillint`
* `bingo get github.com/fatih/faillint@latest`
* `bingo get github.com/fatih/faillint@v1.5.0`
* `bingo get github.com/fatih/faillint@v1.1.0,v1.5.0`

After this, make sure to commit `.bingo` directory in git repository, so the tools will stay versioned! Once pinned, anyone can install correct version of the tool with correct dependencies by either doing:

```bash
bingo get <tool>
```

For example `bingo get faillint`

... or without `bingo`:

```bash
go build -mod=mod -modfile .bingo/<tool>.mod -o=$GOBIN/<tool>-<version>
```

For example `go build -mod=mod -modfile .bingo/faillint.mod -o=$GOBIN/faillint-v1.5.0`

`bingo` allows to easily maintain a separate, nested Go Module for each binary. By default, it will keep it `.bingo/<tool>.mod` This allows to correctly pin the binary without polluting the main go module or other's tool module.

### Using Installed Tools

`bingo get` builds pinned tool or tools in your `$GOBIN` path. Binaries have a name following `<provided-tool-name>-<version>` pattern. So after installation you can do:

* From shell:

```bash
${GOBIN}/<tool>-<version> <args>
```

For example: `${GOBIN}/faillint-v1.5.0`

While it's not the easiest for humans to read or type, it's essential to ensure your scripts use pinned version instead of some non-deterministic "latest version".

> NOTE: If you use `-l` option, bingo creates symlink to <tool> . Use it with care as it's easy to have side effects by having another binary with same name e.g on CI.

`bingo` does not have `run` command [(for a reason)](https://github.com/bwplotka/bingo/issues/52), it provides useful helper variables for script or adhoc use:

> NOTE: Below helpers makes it super easy to install or use pinned binaries without even installing `bingo` (it will use just `go build`!) 💖

* From shell:

```bash
source .bingo/variables.env
${<PROVIDED_TOOL_NAME>} <args>
```

* From Makefile:

```Makefile
include .bingo/Variables.mk
run:
	$(<PROVIDED_TOOL_NAME>) <args>
```

### Real life examples!

Let's show a few, real, sometimes novel examples showcasing `bingo` capabilities:

1. [`golangci-lint`](https://github.com/golangci/golangci-lint) is all-in-one lint framework. It's important to pin it on CI so CI runs are reproducible no matter what new linters are added, removed or changed in new release. Let's pin it to `v1.35.2` and use path recommended by https://golangci-lint.run/usage/install/#install-from-source doc: ` github.com/golangci/golangci-lint/cmd/golangci-lint` (funny enough they discourage `go get` exactly because of the lack of pinning features `bingo` have!)

   ```shell
   bingo get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.35.2
   ```

   This will pin to that commit and install `${GOBIN}/golangci-lint-v1.35.2`

2. It's very common in Go world to use `goimports`, popular `gofmt` replacement which formats Go code including imports. However, not many know that it's breaking compatibility a lot between versions (there are no releases). If you want to assert certain formatting of the Go code in the CI etc your only option is to pin `goimports` version. You can do it via `bingo get`:

   ```shell
   bingo get golang.org/x/tools/cmd/goimports@latest
   ```

   This will install (at the time of writing) latest binary: `${GOBIN}/goimports-v0.0.0-20210112230658-8b4aab62c064`

3. You rather like older formatting? No issue, let's downgrade. Since `goimports` was already installed you can reference it by just `goimports`. Let's pick the commit we want e.g `e64124511800702a4d8d79e04cf6f1af32e7bef2`:

   ```shell
   bingo get goimports@e64124511800702a4d8d79e04cf6f1af32e7bef2
   ```

   This will pin to that commit and install `${GOBIN}/goimports-v0.0.0-20200519204825-e64124511800`

4. Installing (and pinning) multiple versions:

   ```shell
   bingo get goimports@e64124511800702a4d8d79e04cf6f1af32e7bef2,v0.0.0-20200601175630-2caf76543d99,af9456bb636557bdc2b14301a9d48500fdecc053
   ```

   This will pin and install three versions of goimports. Very useful to compatibility testing.

5. Updating to the current latest:

   ```shell
   bingo get goimports@latest
   ```

   This will find the latest module version, pin and install it.

6. Listing binaries you have pinned:

   ```shell
   bingo list
   ```

7. Unpinning `goimports` totally from the project:

   ```shell
   bingo get goimports@none
   ```

   > PS: `go get` also allows `@none` suffix! Did you know? I didn't (:*

8. Installing all tools:

   ```shell
   bingo get
   ```

9. **Bonus**: Have you ever dreamed to pin command from bigger project like... `thanos`? I was. Can you even install it using Go tooling? Let's try:

   ```shell
   go get github.com/thanos-io/thanos/cmd/thanos@v0.17.2
   # Output: go: cannot use path@version syntax in GOPATH mode
   ```

   Ups you cannot use this in non-Go project at all... (: Let's create setup go mod and retry:

   ```shell
   go mod init _
   # Output: go: creating new go.mod: module 
   go get github.com/thanos-io/thanos/cmd/thanos@v0.17.2
   # go get github.com/thanos-io/thanos/cmd/thanos@v0.17.2
   # go: downloading github.com/thanos-io/thanos v0.17.2
   # go: found github.com/thanos-io/thanos/cmd/thanos in github.com/thanos-io/thanos v0.17.2
   # go get: github.com/thanos-io/thanos@v0.17.2 requires
   # github.com/cortexproject/cortex@v1.5.1-0.20201111110551-ba512881b076 requires
   # github.com/thanos-io/thanos@v0.13.1-0.20201030101306-47f9a225cc52 requires
   # github.com/cortexproject/cortex@v1.4.1-0.20201030080541-83ad6df2abea requires
   # github.com/thanos-io/thanos@v0.13.1-0.20201019130456-f41940581d9a requires
   # github.com/cortexproject/cortex@v1.3.1-0.20200923145333-8587ea61fe17 requires
   # github.com/thanos-io/thanos@v0.13.1-0.20200807203500-9b578afb4763 requires
   # github.com/cortexproject/cortex@v1.2.1-0.20200805064754-d8edc95e2c91 requires
   # github.com/thanos-io/thanos@v0.13.1-0.20200731083140-69b87607decf requires
   # github.com/cortexproject/cortex@v0.6.1-0.20200228110116-92ab6cbe0995 requires
   # github.com/prometheus/alertmanager@v0.19.0 requires
   # github.com/prometheus/prometheus@v0.0.0-20190818123050-43acd0e2e93f requires
   # k8s.io/client-go@v12.0.0+incompatible: reading https://proxy.golang.org/k8s.io/client-go/@v/v12.0.0+incompatible.mod: 410 Gone
   # server response: not found: k8s.io/client-go@v12.0.0+incompatible: invalid version: +incompatible suffix not allowed: module contains a go.mod file, so semantic import versioning is required
   ```

   The reasoning is complex but [TL;DR: Go Modules are just sometimes hard to be properly used for some projects](https://twitter.com/bwplotka/status/1347104281120403458). This is why bigger projects like `Kubernetes`, `Prometheus` or `Thanos` has to use `replace` statements (plus others like `exclude` or `retract`). To make this `go get` work we would need to manually craft `replace` statements in our own go `mod` file. But what if we don't want to do that or don't know how or simply we want to install pinned version of Thanos locally without having Go project? Just use bingo:

   ```shell
   bingo get github.com/thanos-io/thanos/cmd/thanos@v0.17.2
   ${GOBIN}/thanos-v0.17.2 --help
   ```

## Advanced Techniques

* Using advanced go build flags and environment variables.

To tell bingo to use certain env vars and tags during build time, just add them as a comment to the go.mod file manually and do `bingo get`. Done!

NOTE: Order of comment matters. First bingo expects relative package name (optional), then environment variables, then flags. All space delimited.

Real example from production project that relies on extended Hugo.

```
module _ // Auto generated by https://github.com/bwplotka/bingo. DO NOT EDIT

go 1.16

require github.com/gohugoio/hugo v0.83.1 // CGO_ENABLED=1 -tags=extended
```

Run `bingo list` to see if build options are parsed correctly. Run `bingo get` to install all binaries including the modified one with new build flags.

## Production Usage

To see production example see:

* [bingo's own tools](https://github.com/bwplotka/bingo/tree/master/.bingo)
* [Thanos's tools](https://github.com/thanos-io/thanos/tree/7bf3b0f8f3af57ac3aef033f6efb58860f273c78/.bingo)
* [go-grpc-middleware's tools](https://github.com/grpc-ecosystem/go-grpc-middleware/tree/5b83c99199db53d4258b05646007b48e4658b3af/.bingo)

## Contributing

Any contributions are welcome! Just use GitHub Issues and Pull Requests as usual. We follow [Thanos Go coding style](https://thanos.io/tip/contributing/coding-style-guide.md/) guide.

See an extensive and up-to-date description of the `bingo` usage below:

## Command Help

```bash mdox-exec="bingo --help" mdox-expect-exit-code=2
bingo: 'go get' like, simple CLI that allows automated versioning of Go package level binaries (e.g required as dev tools by your project!)
built on top of Go Modules, allowing reproducible dev environments. 'bingo' allows to easily maintain a separate, nested Go Module for each binary.

For detailed examples and documentation see: https://github.com/bwplotka/bingo

'bingo' supports following commands:

Commands:

  get <flags> [<package or binary>[@version1,none,latest,version2,version3...]]

  -go string
    	Path to the go command. (default "go")
  -insecure
    	Use -insecure flag when using 'go get'
  -l	If enabled, bingo will also create soft link called <tool> that links to the current<tool>-<version> binary. Use Variables.mk and variables.env if you want to be sure that what you are invoking is what is pinned.
  -moddir string
    	Directory where separate modules for each binary will be maintained. Feel free to commit this directory to your VCS to bond binary versions to your project code. If the directory does not exist bingo logs and assumes a fresh project. (default ".bingo")
  -n string
    	The -n flag instructs to get binary and name it with given name instead of default, so the last element of package directory. Allowed characters [A-z0-9._-]. If -n is used and no package/binary is specified, bingo get will return error. If -n is used with existing binary name, copy of this binary will be done. Cannot be used with -r
  -r string
    	The -r flag instructs to get existing binary and rename it with given name. Allowed characters [A-z0-9._-]. If -r is used and no package/binary is specified or non existing binary name is used, bingo will return error. Cannot be used with -n.
  -v	Print more'


  list <flags> [<package or binary>]

List enumerates all or one binary that are/is currently pinned in this project. It will print exact path, Version and immutable output.

  -moddir string
    	Directory where separate modules for each binary is maintained. If does not exists, bingo list will fail. (default ".bingo")
  -v	Print more'


  version

Prints bingo Version.
```

## Initial Author

[@bwplotka](https://bwplotka.dev) inspired by [Paul's](https://github.com/myitcv) research and with a bit of help from [Duco](https://github.com/Helcaraxan) (:
