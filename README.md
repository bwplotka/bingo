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
* No need to install `bingo` in order to **use** pinned tools. This avoids the `chicken & egg` problem. Only `go build` required.
* Easy upgrade, downgrade, addition, or removal of the needed binary's version, with no risk of dependency conflicts.
  * NOTE: Tools are **often** not following semantic versioning, so `bingo` allows to pin by the commit.
* Immutable binary names, which gives a reliable way for users and CIs to use the expected version of the binaries, with reinstall on-demand only if needed.
* Works with all buildable Go projects, including pre Go modules and complex projects with complex `replace` statements. (e.g Kuberentes)
* Optional, automatic integration with Makefiles.

You can read full a story behind `bingo` [in this blog post](https://deploy-preview-16--bwplotka.netlify.app/2020/bingo/).

## Requirements

* Go 1.14+
* Linux or MacOS (Want Windows support? [Helps us out](https://github.com/bwplotka/bingo/issues/26), should be trivial!)
* All tools that you wish to "pin" have to be built in Go (they don't need to use Go modules at all).

## Installing

In your repository (does not need to be a Go project)

```shell
go get -u github.com/bwplotka/bingo
```

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

Once pinned, anyone can install correct version of the tool with correct dependencies by either doing:

```bash
go build -modfile .bingo/<tool>.mod -o=$GOBIN/<tool>-<version>
```

or

```bash
bingo get <tool>
```

`bingo` allows to easily maintain a separate, nested Go Module for each binary. By default, it will keep it `.bingo/<tool>.mod` This allows to correctly pin the binary without polluting the main go module or other's tool module.

### Using Installed Tools

`bingo get` builds pinned tool or tools in your `$GOBIN` path. Binaries have a name following `<provided-tool-name>-<version>` pattern. So after installation 
you can do:

* From shell:

```bash
${GOBIN}/<tool>-<version> <args>
```

While it's not the easiest for humans to read or type, it's essential to ensure your scripts use pinned version instead of some "unknown" latest version.

> NOTE: If you use `-l` option, bingo creates symlink to <tool>. Use it with care as it's easy to have side effects by having another binary with same name e.g on CI. 

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

1. It's very common in Go world to use `goimports`, popular `gofmt` replacement which formats Go code including imports. However, not many know that it's breaking compatibility a lot between versions (there are no releases). If you want to assert certain formatting of the Go code in the CI etc your only option is to pin `goimports` version. You can do it via `bingo get`:

   ```shell
   bingo get -u golang.org/x/tools/cmd/goimports
   ```

   This will install (at the time of writing) binary: `${GOBIN}/goimports-v0.0.0-20200601175630-2caf76543d99`

2. After running above, pinning (or downgrading/upgrading) version:

   ```shell
   bingo get goimports@e64124511800702a4d8d79e04cf6f1af32e7bef2
   ```

   This will pin to that commit and install `${GOBIN}/goimports-v0.0.0-20200519204825-e64124511800`

3. Installing (and pinning) multiple versions:

   ```shell
   bingo get goimports@e64124511800702a4d8d79e04cf6f1af32e7bef2,v0.0.0-20200601175630-2caf76543d99,af9456bb636557bdc2b14301a9d48500fdecc053
   ```

   This will pin and install three versions of goimports. Very useful to compatibility testing.

4. Unpinning `goimports` totally from the project:

   ```shell
   bingo get goimports@none
   ```

   *PS: `go get` allows that, did you know? I didn't (:*

5. Editing `.mod` file manually. You can totally go to `.bingo/goimports.mod` and edit the version manually. Just make sure to `bingo get goimports` to install that version!

6. Installing all tools:

   ```shell
   bingo get
   ```

7. Using `goimports`

   ```bash
   ${GOBIN}/goimports-v0.0.0-20200519204825-e64124511800
   ```

8. Using `goimports` while not hard-coding version:

* From shell:

  ```bash
  source .bingo/variables.env
  ${GOIMPORTS} <args>
  ```

* From Makefile:

```Makefile
include .bingo/Variables.mk
run:
	@$(GOIMPORTS) <args>
```

1. Bonus: Makefile mode! If you use `Makefile` , `bingo` generates a very simple helper with nice variables. After running any `bingo get` command, you will notice`.bingo/Variables.mk` file. Feel free to include this in your Makefile (`include .bingo/Variables.mk` on the top of your Makefile).

   From now in your Makefile you can use, e.g. `$(GOIMPORTS)` variable which reliably ensures a correct version is used and installed.

2. Bonus number 2! Using immutable names might be hard to maintain for your other scripts so `bingo` also produces environment variables you can source to you shell. It's as easy as:

   ```shell
   source .bingo/variables.env
   ```

   From now on you can use, e.g. `$(GOIMPORTS)` variable which holds currently pinned binary name of the goimports tool.

## Production Usage

To see production example see:

* [bingo's own tools](https://github.com/bwplotka/bingo/tree/master/.bingo)
* [Thanos's tools](https://github.com/thanos-io/thanos/tree/7bf3b0f8f3af57ac3aef033f6efb58860f273c78/.bingo)
* [go-grpc-middleware's tools](https://github.com/grpc-ecosystem/go-grpc-middleware/tree/5b83c99199db53d4258b05646007b48e4658b3af/.bingo)

## Contributing

Any contributions are welcome! Just use GitHub Issues and Pull Requests as usual. We follow [Thanos Go coding style](https://thanos.io/tip/contributing/coding-style-guide.md/) guide.

See an extensive and up-to-date description of the `bingo` usage below:

## Command Help

```bash mdox-gen-exec="sh -c 'bingo --help || exit 0'"
bingo: 'go get' like, simple CLI that allows automated versioning of Go package level binaries (e.g required as dev tools by your project!)
built on top of Go Modules, allowing reproducible dev environments. 'bingo' allows to easily maintain a separate, nested Go Module for each binary.

For detailed examples see: https://github.com/bwplotka/bingo

'bingo' supports following commands:

Commands:

  get <flags> [<package or binary>[@version1 or none,version2,version3...]]

Similar to 'go get' you can pull, install and pin required 'main' (buildable Go) package as your tool in your project.

'bingo get <repo/org/tool>' will resolve given main package path, download it using 'go get -d', then will produce directory (controlled by -moddir flag) and put
separate, specially commented module called <tool>.mod. After that, it installs given package as '$GOBIN/<tool>-<Version>'.

Once installed at least once, 'get' allows to reference the tool via it's name (without Version) to install, downgrade, upgrade or remove.
Similar to 'go get' you can get binary with given Version: a git commit, git tag or Go Modules pseudo Version after @:

'bingo get <repo/org/tool>@<Version>' or 'bingo get <tool>@<Version>'

'get' without any argument will download and get ALL the tools in the moddir directory.
'get' also allows bulk pinning and install. Just specify multiple versions after '@':

'bingo get <tool>@<version1,version2,tag3>'

Similar to 'go get' you can use -u and -u=patch to control update logic and '@none' to remove binary.

Once pinned apart of 'bingo get', you can also use 'go build -modfile .bingo/<tool>.mod -o=<where you want to build> <tool package>' to install
correct Version of a tool.

Note that 'bingo' creates additional useful files inside -moddir:

* '<moddir>/Variables.mk': When included in your Makefile ('include <moddir>/Variables.mk'), you can refer to each binary
using '$(TOOL)' variable. It will also  install correct Version if missing.
* '<moddir>/variables.env': When sourced ('source <moddir>/variables.env') you can refer to each binary using '$(TOOL)' variable.
It will NOT install correct Version if missing.

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
  -u	The -u flag instructs get to update modules providing dependencies of packages named on the command line to use newer minor or patch releases when available.
  -upatch
    	The -upatch flag (not -u patch) also instructs get to update dependencies, but changes the default to select patch releases.
  -v	Print more'


  list <flags> [<package or binary>]

List enumerates all or one binary that are/is currently pinned in this project. It will print exact path, Version and immutable output.

  -moddir string
    	Directory where separate modules for each binary is maintained. If does not exists, bingo list will fail. (default ".bingo")
  -v	Print more'


  Version

Prints bingo Version.
Error: Failed to parse flags: flag: help requested

bingo: 'go get' like, simple CLI that allows automated versioning of Go package level binaries (e.g required as dev tools by your project!)
built on top of Go Modules, allowing reproducible dev environments. 'bingo' allows to easily maintain a separate, nested Go Module for each binary.

For detailed examples see: https://github.com/bwplotka/bingo

'bingo' supports following commands:

Commands:

  get <flags> [<package or binary>[@version1 or none,version2,version3...]]

Similar to 'go get' you can pull, install and pin required 'main' (buildable Go) package as your tool in your project.

'bingo get <repo/org/tool>' will resolve given main package path, download it using 'go get -d', then will produce directory (controlled by -moddir flag) and put
separate, specially commented module called <tool>.mod. After that, it installs given package as '$GOBIN/<tool>-<Version>'.

Once installed at least once, 'get' allows to reference the tool via it's name (without Version) to install, downgrade, upgrade or remove.
Similar to 'go get' you can get binary with given Version: a git commit, git tag or Go Modules pseudo Version after @:

'bingo get <repo/org/tool>@<Version>' or 'bingo get <tool>@<Version>'

'get' without any argument will download and get ALL the tools in the moddir directory.
'get' also allows bulk pinning and install. Just specify multiple versions after '@':

'bingo get <tool>@<version1,version2,tag3>'

Similar to 'go get' you can use -u and -u=patch to control update logic and '@none' to remove binary.

Once pinned apart of 'bingo get', you can also use 'go build -modfile .bingo/<tool>.mod -o=<where you want to build> <tool package>' to install
correct Version of a tool.

Note that 'bingo' creates additional useful files inside -moddir:

* '<moddir>/Variables.mk': When included in your Makefile ('include <moddir>/Variables.mk'), you can refer to each binary
using '$(TOOL)' variable. It will also  install correct Version if missing.
* '<moddir>/variables.env': When sourced ('source <moddir>/variables.env') you can refer to each binary using '$(TOOL)' variable.
It will NOT install correct Version if missing.

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
  -u	The -u flag instructs get to update modules providing dependencies of packages named on the command line to use newer minor or patch releases when available.
  -upatch
    	The -upatch flag (not -u patch) also instructs get to update dependencies, but changes the default to select patch releases.
  -v	Print more'


  list <flags> [<package or binary>]

List enumerates all or one binary that are/is currently pinned in this project. It will print exact path, Version and immutable output.

  -moddir string
    	Directory where separate modules for each binary is maintained. If does not exists, bingo list will fail. (default ".bingo")
  -v	Print more'


  Version

Prints bingo Version.
```

## Initial Author

[@bwplotka](https://bwplotka.dev) inspired by [Paul's](https://github.com/myitcv) research and with a bit of help from [Duco](https://github.com/Helcaraxan) (:
