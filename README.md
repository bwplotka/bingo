# bingo
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/bwplotka/bingo)
[![Latest Release](https://img.shields.io/github/release/bwplotka/bingo.svg?style=flat-square)](https://github.com/bwplotka/bingo/releases/latest)
[![CI](https://github.com/bwplotka/bingo/workflows/go/badge.svg)](https://github.com/bwplotka/bingo/actions?query=workflow%3Ago)
[![Go Report Card](https://goreportcard.com/badge/github.com/bwplotka/bingo)](https://goreportcard.com/report/github.com/bwplotka/bingo)

`go get` like, simple CLI that allows automated versioning of Go package level binaries (e.g required as dev tools by your project!)
built on top of Go Modules, allowing reproducible dev environments.

[![Demo](examples/bingo-demo.gif)](examples/)

## Features

From our experience all repositories and projects require some tools and binaries to be present on the machine to be able to perform various development
operations like building, formatting, releasing or static analysis. For smooth development all such tools should be pinned to a certain version and
bounded to the code commits there were meant to be used against.

Go modules does not aim to solve this problem, and even if they will do at some point it will not be on the package level, which makes it impossible to e.g
pin minor version `X.Y.0` of package `module1/cmd/abc` and version `X.Z.0` of `module1/cmd/def`.

At the end `bingo`, has following features:

* It allows maintaining separate, hidden, nested Go modules for Go buildable packages you need **without obfuscating your own module or worrying with tool's cross dependencies**!
* Package level versioning, which allows versioning different (or the same!) package multiple times from a single module in different versions.
* Works also for non-Go projects. It only requires the tools to be written in Go.
* No need to install `bingo` in order to **use** pinned tools. This avoids the `chicken & egg` problem. Only `go build` required.
* Easy upgrade, downgrade, addition, or removal of the needed binary's version, with no risk of dependency conflicts.
    * NOTE: Tools are **often** not following semantic versioning, so `bingo` allows to pin by the commit.
* Immutable binary names, which gives a reliable way for users and CIs to use the expected version of the binaries, with reinstall on-demand only if needed.
* Optional, automatic integration with Makefiles.

You can read full a story behind `bingo` [in this blog post](https://deploy-preview-16--bwplotka.netlify.app/2020/bingo/).

## Requirements

* Go 1.14+
* Linux or MacOS.
* All tools that you wish to "pin" have to be build in Go and use [Go Modules].

## Installing

```shell
go get github.com/bwplotka/bingo && go mod tidy
```

or if you already installed bingo and want to pin it (inception!):

```shell
bingo get -u github.com/bwplotka/bingo
```

## Usage

The key idea is that you can manage your tools similar to your Go dependencies via `go get`:

```shell
bingo get [<package or binary>[@version1 or none,version2,version3...]]
```

Once pinned, anyone can reliably install correct version of the tool either doing:

```bash
go build -modfile .bingo/<tool>.mod -o=<where you want to build> <tool package>
```

or

```bash
bingo get <tool>
```

Also, make sure to check out the `-m` option if your project uses `Makefile` to generate useful helper variables 💖that makes it super easy to install pinned
binaries without even installing `bingo` (it will use just `go build`).

`bingo` allows to easily maintain a separate, nested Go Module for each binary. By default, it will keep it `.bingo/<tool>.mod`
This allows to correctly pin the binary without polluting the main go module or other's tool module.

See an extensive and up-to-date description of the `bingo` usage below:

[embedmd]:# (bingo-help.txt $)
```$
bingo: 'go get' like, simple CLI that allows automated versioning of Go package level binaries (e.g required as dev tools by your project!)
built on top of Go Modules, allowing reproducible dev environments.

The key idea is that 'bingo' allows to easily maintain a separate, nested Go Module for each binary. By default, it will keep it '.bingo/<tool>.mod'
This allows to correctly pin the tool without polluting the main go module or other's tool module.

For detailed examples see: https://github.com/bwplotka/bingo

'bingo' supports following commands:

Commands:

  get <flags> [<package or binary>[@version1 or none,version2,version3...]]

Similar to 'go get' you can pull, install and pin required 'main' (buildable Go) package as your tool in your project.

'bingo get <repo/org/tool>' will resolve given main package path, download it using 'go get -d', then will produce directory (controlled by -moddir flag) and put
separate, specially commented module called <tool>.mod. After that, it installs given package as '$GOBIN/<tool>-<version>'.

Once installed at least once, 'get' allows to reference the tool via it's name (without version) to install, downgrade, upgrade or remove.
Similar to 'go get' you can get binary with given version: a git commit, git tag or Go Modules pseudo version after @:

'bingo get <repo/org/tool>@<version>' or 'bingo get <tool>@<version>'

'get' without any argument will download and get ALL the tools in the moddir directory.
'get' also allows bulk pinning and install. Just specify multiple versions after '@':

'bingo get <tool>@<version1,version2,tag3>'

Similar to 'go get' you can use -u and -u=patch to control update logic and '@none' to remove binary.

Once pinned apart of 'bingo get', you can also use 'go build -modfile .bingo/<tool>.mod -o=<where you want to build> <tool package>' to install
correct version of a tool.

'-m' option creates '<moddir>/Variables.mk' and attempts to include this in your own 'Makefile'.

Thanks to that you can refer to the binary using '$(TOOL)' variable which will install correct version if missing.


  -go string
    	Path to the go command. (default "go")
  -insecure
    	Use -insecure flag when using 'go get'
  -m	Generate makefile helper with all binaries as variables.
  -makefile -m
    	Makefile to link the the generated helper for make when -m options is specified with.Specify empty to disable including the helper. (default "Makefile")
  -moddir string
    	Directory where separate modules for each binary will be maintained. Feel free to commit this directory to your VCS to bond binary versions to your project code. If the directorydoes not exist bingo logs and assumes a fresh project. (default ".bingo")
  -n string
    	The -n flag instructs to get binary and name it with given name instead of default, so the last element of package directory Allowed characters [A-z0-9._-]. If -n is used and no package/binary is specified, bingo get will return error. If -n is used with existing binary name, rename will be done.
  -u	The -u flag instructs get to update modules providing dependencies of packages named on the command line to use newer minor or patch releases when available.
  -upatch
    	The -upatch flag (not -u patch) also instructs get to update dependencies, but changes the default to select patch releases.
  -v	Print more'


  list <flags> [<package or binary>]

List enumerates all or one binary that are/is currently pinned in this project. It will print exact path, version and immutable output.

  -moddir string
    	Directory where separate modules for each binary ismaintained. If does not exists, bingo list will fail. (default ".bingo")
  -v	Print more'


  version

Prints bingo version.

```

## Examples:

Let's show a few examples on popular `goimports` tool (which formats Go code including imports):

1. Pinning latest `goimports`:

    ```shell
    bingo -u get golang.org/x/tools/cmd/goimports
    ```

    This will install (at the time of writing) binary: `${GOBIN}/goimports-v0.0.0-20200601175630-2caf76543d99`

1. After running above, pinning (or downgrading/upgrading) version:

    ```shell
    bingo get goimports@e64124511800702a4d8d79e04cf6f1af32e7bef2
    ```

    This will pin to that commit and install `${GOBIN}/goimports-v0.0.0-20200519204825-e64124511800`

1. Installing (and pinning) multiple versions:

    ```shell
    bingo get goimports@e64124511800702a4d8d79e04cf6f1af32e7bef2,v0.0.0-20200601175630-2caf76543d99,af9456bb636557bdc2b14301a9d48500fdecc053
    ```

    This will pin and install three versions of goimports. Very useful to compatibility testing.

1. Unpinning `goimports` totally from the project:

    ```shell
    bingo get goimports@none
    ```

    _PS: `go get` allows that, did you know? I didn't (:_

1. Editing `.mod` file manually. You can totally go to `.bingo/goimports.mod` and edit the version manually. Just make sure to `bingo get goimports` to install that version!

1. Installing all tools:

    ```shell
    bingo get
    ```

1. Makefile mode! If you use `Makefile` , bingo allows you to generate a very simple helper with nice variables. Just run the same commands as above, but with
`-m` option e.g:

    ```shell
    bingo get -m goimports
    ```

    This will produce `.bingo/Variables.mk` and attempt to `include` that in your main `Makefile`.

    From now on you can use `$(GOIMPORTS)` variable which reliably ensures a correct version is used and installed.

## Production Usage

To see production example see:

 * [bingo's own tools](https://github.com/bwplotka/bingo/tree/master/.bingo)
 * [Thanos's tools](https://github.com/thanos-io/thanos/tree/master/.bingo)
 * [go-grpc-middleware's tools](https://github.com/grpc-ecosystem/go-grpc-middleware/tree/5b83c99199db53d4258b05646007b48e4658b3af/.bingo)

## Contributing

Any contributions are welcome! Just use GitHub Issues and Pull Requests as usual.
We follow [Thanos Go coding style](https://thanos.io/contributing/coding-style-guide.md/) guide.

## Initial Author

[@bwplotka](https://bwplotka.dev)
