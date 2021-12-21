# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/) and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

NOTE: As semantic versioning states all 0.y.z releases can contain breaking changes in API (flags, grpc API, any backward compatibility)

We use *breaking* word for marking changes that are not backward compatible (relates only to v0.y.z releases.)

## Unreleased

## [v0.5.2](https://github.com/bwplotka/bingo/releases/tag/v0.5.2) - 2021.12.21

### Added

* Added back go sum files for each pinned go tool for extra security (check sums).

### Fixed

* Fixed support for modules / packages with upper case in it and Go 1.17 logic.
* Fixed support for modules with +incompatible version.

## [v0.5.1](https://github.com/bwplotka/bingo/releases/tag/v0.5.1) - 2021.07.20

### Added

* bingo now auto-fetches `exclude` and `retract` directives from binaries' go modules that have those.

### Changed

* *breaking*: Use `bingo:no_directive_fetch` to disable auto fetch logic (previously: `bingo:no_replace_fetch`)

## [v0.4.3](https://github.com/bwplotka/bingo/releases/tag/v0.4.3) - 2021.05.14

### Fixed

* Fixed panic when calling wrong version with short module name.
* Fixed issue when installing bingo from scratch using `go get`

## [v0.4.2](https://github.com/bwplotka/bingo/releases/tag/v0.4.2) - 2021.05.13

### Added

* bingo list now lists pinned build flags and environment variables.

### Fixed

* Fixed preserving build flags and env files via bingo .mod files.
* Fixed formatting of bingo list

## [v0.4.1](https://github.com/bwplotka/bingo/releases/tag/v0.4.1) - 2021.05.12

### Added

* Added support for build flags and environment variables via go.mod file.

### Fixed

* Generated files have limited permission.
* Support for Go pre-released versions.

## [v0.4.0](https://github.com/bwplotka/bingo/releases/tag/v0.4.0) - 2021.03.24

### Added

* Added support for Go 1.16, following the changes it introduces in the module system: https://blog.golang.org/go116-module-changes.

## [v0.3.1](https://github.com/bwplotka/bingo/releases/tag/v0.3.1) - 2021.02.02

### Fixed

* [Fixed](https://github.com/bwplotka/bingo/issues/65) support for tools with names that have capital letters.

## [v0.3.0](https://github.com/bwplotka/bingo/releases/tag/v0.3.0) - 2021.01.13

### Added

* `-l` flag which also creates a soft link to the currently pinned tool under non versioned <tool> binary name.
* Support easier path changing upgrades of tools with the same name.
* [Automatic download of `replace` entries](https://github.com/bwplotka/bingo/issues/7) for the pinned version of the tool. This is very often required by big projects to fight with deps hell ([Go Modules are hard](https://twitter.com/bwplotka/status/1347104281120403458)). Add `// bingo:no_replace_fetch` comment anywhere in tool mod file if you want to not autogenerate replace commands.

### Fixed

* Simplified and fixed -u cases.
* Fixed various invalid cases for -r and -n options.
* Extended capabilities of verbose mode.

## [v0.2.4](https://github.com/bwplotka/bingo/releases/tag/v0.2.4) - 2020.12.27

### Fixed

* Improved env variables
* Removed -i option from build which was not needed.
* Avoid vendor mode when installing via Makefile

## [v0.2.3](https://github.com/bwplotka/bingo/releases/tag/v0.2.3) - 2020.06.26

### Fixed

* Fixed Go version checker.
* Fixed case with installing latest binary version, when binary was not installed before.

## [v0.2.2](https://github.com/bwplotka/bingo/releases/tag/v0.2.2) - 2020.06.10

### Fixed

* [#25](https://github.com/bwplotka/bingo/issues/25) Fixed support of `bingo get` for arrays.
* Fixed versioning binaries with `+incompatible` version (wrong templating used).
* Fixed support `bingo list` for arrays.
* Added rename / clone logic
* Always print to stdout no matter of verbose level.

### Changed

* `bingo list` output format. (table `\t`-delimited now)

## [v0.2.1](https://github.com/bwplotka/bingo/releases/tag/v0.2.1) - 2020.06.04

### Fixed

* Fixed extra whitespace in variables.env.

## [v0.2.0](https://github.com/bwplotka/bingo/releases/tag/v0.2.0) - 2020.06.04

### Added

* Added `.variables.env` file to bingo moddir for easy export of all environment variables to the current shell. Removed `-m` and `-makefile` flags. Bingo now always creates makefile and env file and never generate `include` to avoid many corner cases. It's now documented how to add `include` in the documentation.

## [v0.1.1](https://github.com/bwplotka/bingo/releases/tag/v0.1.1) - 2020.06.03

### Fixed

* [#22](https://github.com/bwplotka/bingo/pull/22) Fixed problem with running bingo in non-Go project. From now on it also maintains fake go.mod to resolve issues like:

```
`Error: get command failed: 0: getting : go get -d: go: cannot find main module, but -modfile was set.
	-modfile cannot be used to set the module root directory.
```

## [v0.1.0](https://github.com/bwplotka/bingo/releases/tag/v0.1.0) - 2020.05.30

Initial release.

Why 0.1.0? Well, because we plan to release 1.0 once we introduce this tool to [Thanos](http://github.com/thanos-io/thanos) and [go-grpc-middleware](https://github.com/grpc-ecosystem/go-grpc-middleware) as the final test (: After having this usage stable for a bit, and we are sure flags will not change, we can claim 1.0.
